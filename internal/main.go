package internal

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/florianloch/cassette/internal/constants"
	"github.com/florianloch/cassette/internal/handler"
	"github.com/florianloch/cassette/internal/middleware"
	"github.com/florianloch/cassette/internal/persistence"
	"github.com/florianloch/cassette/internal/spotify"
	"github.com/florianloch/cassette/internal/util"

	"github.com/NYTimes/gziphandler"
	"github.com/go-chi/chi"
	chiMiddleware "github.com/go-chi/chi/middleware"
	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	spotifyAPI "github.com/zmb3/spotify"
	"golang.org/x/oauth2"
)

var (
	redirectURL *url.URL
	auth        spotify.SpotAuthenticator
	store       *sessions.CookieStore
	dao         *persistence.PlayerStatesDAO
	// createSpotClient is required to use different initilisation code for testing
	// for production environment
	createSpotClient     spotClientCreator
	isDevMode            bool
	webStaticContentPath = "/web/dist"
)

type spotClientCreator func(token *oauth2.Token) spotify.SpotClient
type m map[string]interface{}

func RunInProduction() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	isDevMode = util.Env(constants.EnvENV, "") == "DEV"
	if isDevMode {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("Running in DEV mode. Being more verbose. Set environment variable 'ENV' to 'DEV' to activate.")
	}

	networkInterface := util.Env(constants.EnvNetworkInterface, constants.DefaultNetworkInterface)
	// We also have to check for "PORT" as that is how Heroku/Dokku etc. tells the app where to listen
	port := util.Env(constants.EnvPort, os.Getenv("PORT"))
	appURL := util.Env(constants.EnvAppURL, "http://"+networkInterface+":"+port+"/")

	secret32Bytes, err := util.Make32ByteSecret(util.Env(constants.EnvSecret, ""))
	if err != nil {
		log.Fatal().Err(err).Msg("Could not generate secret. Aborting.")
	}

	mongoDBURI := util.Env(constants.EnvMongoURI, "")
	if mongoDBURI == "" {
		log.Fatal().Msg("No URI for connecting to MongoDB given. Aborting.")
	}
	dao, err = persistence.Connect(mongoDBURI)
	if err != nil {
		log.Fatal().Err(err).Str("mongoDBURI", mongoDBURI).Msg("Failed connecting to MongoDB.")
	}

	store = sessions.NewCookieStore(secret32Bytes)
	store.Options.HttpOnly = true
	store.Options.Secure = !isDevMode
	store.Options.SameSite = http.SameSiteLaxMode

	redirectURL, err = url.Parse(appURL)
	if err != nil {
		log.Fatal().Err(err).Str("appURL", appURL).Msgf("'%s' variable is not set to a valid value.", constants.EnvAppURL)
	}
	redirectURL.Path = "/spotify-oauth-callback"

	tmp := spotifyAPI.NewAuthenticator(redirectURL.String(), spotifyAPI.ScopeUserReadCurrentlyPlaying, spotifyAPI.ScopeUserReadPlaybackState, spotifyAPI.ScopeUserModifyPlaybackState)
	auth = &tmp

	createSpotClient = func(token *oauth2.Token) spotify.SpotClient {
		client := auth.NewClient(token)
		return &client
	}

	r := setupAPI(secret32Bytes)

	var interfacePort = networkInterface + ":" + port
	log.Info().Msgf("Webserver started on %s", interfacePort)

	err = http.ListenAndServe(interfacePort, r)
	log.Fatal().Err(err).Msg("Server terminated.")
}

func SetupForTest(mockedAuth spotify.SpotAuthenticator, spotClientMockCreator spotClientCreator) http.Handler {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)

	isDevMode = util.Env(constants.EnvENV, "") == "DEV"
	if isDevMode {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("Running in DEV mode. Being more verbose. Set environment variable 'ENV' to 'DEV' to activate.")
	}

	secret32Bytes, err := util.Make32ByteSecret(util.Env(constants.EnvSecret, ""))
	if err != nil {
		log.Fatal().Err(err).Msg("Could not generate secret. Aborting.")
	}

	mongoDBURI := util.Env(constants.EnvMongoURI, "")
	if mongoDBURI == "" {
		log.Fatal().Msg("No URI for connecting to MongoDB given. Aborting.")
	}
	dao, err = persistence.Connect(mongoDBURI)
	if err != nil {
		log.Fatal().Err(err).Str("mongoDBURI", mongoDBURI).Msg("Failed connecting to MongoDB.")
	}

	store = sessions.NewCookieStore(secret32Bytes)

	auth = mockedAuth

	createSpotClient = spotClientMockCreator

	return setupAPI(secret32Bytes)
}

func setupAPI(secret32Bytes []byte) http.Handler {
	gob.Register(&spotifyAPI.PrivateUser{})
	gob.Register(&oauth2.Token{})
	gob.Register(&m{})

	var clientID = util.Env(constants.EnvSpotifyClientID, "")
	var clientSecret = util.Env(constants.EnvSpotifyClientSecret, "")

	if clientID == "" || clientSecret == "" {
		log.Fatal().Msgf("Please make sure '%s' and '%s' are set. Aborting.", constants.EnvSpotifyClientID, constants.EnvSpotifyClientSecret)
	}

	auth.SetAuthInfo(clientID, clientSecret)

	var csrfMiddleware = csrf.Protect(
		secret32Bytes,
		csrf.RequestHeader(constants.CSRFHeaderName),
		csrf.CookieName(constants.CSRFCookieName),
		csrf.Secure(!isDevMode),
		csrf.HttpOnly(true),
		csrf.SameSite(csrf.SameSiteLaxMode),
		csrf.MaxAge(60*60*24*365), // Cookie is valid for 1 year
		csrf.ErrorHandler(csrfErrorHandler{}),
	)

	spotAuthMiddleware, spotOAuthCBHandler := middleware.CreateSpotifyAuthMiddleware(auth, redirectURL)

	var cwd, _ = os.Getwd()
	var staticAssetsPath = cwd + webStaticContentPath
	var spaHandler = handler.
		NewSpaHandler(staticAssetsPath, "index.html").
		SetFileServer(gziphandler.GzipHandler(http.FileServer(http.Dir(staticAssetsPath))))
	log.Info().Msgf("Loading assets from: '%s'", staticAssetsPath)

	r := chi.NewRouter()

	r.Use(chiMiddleware.RequestID)
	r.Use(hlog.NewHandler(log.Logger))
	r.Use(middleware.ChiRequestIDHandler("reqID", ""))
	r.Use(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		hlog.FromRequest(r).Info().
			Dur("dur(ms)", duration).
			Int("size(bytes)", size).
			Int("status", status).
			Stringer("url", r.URL).
			Str("verb", r.Method).
			Msg("")
	}))
	r.Use(chiMiddleware.Recoverer)

	r.Use(attachSession)

	r.Get(redirectURL.Path, spotOAuthCBHandler)

	r.Route("/api", func(r chi.Router) {
		// if isDevMode {
		// 	r.Use(debugLogger)
		// }
		r.Use(csrfMiddleware)

		r.Head("/csrfToken", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(constants.CSRFHeaderName, csrf.Token(r))
		})

		r.With(attachDAO).With(attachUser).Route("/you", func(r chi.Router) {
			r.Get("/", handler.UserExportHandler)
			r.Delete("/", handler.UserDeleteHandler)
		})

		r.With(attachSpotifyClient).Get("/activeDevices", handler.ActiveDevicesHandler)

		r.With(attachSpotifyClient).With(attachDAO).With(attachUser).Route("/playerStates", func(r chi.Router) {
			r.Post("/", handler.StorePostHandler)
			r.Get("/", handler.StoreGetHandler)
			r.With(attachSlot).Route("/{slot}", func(r chi.Router) {
				r.Put("/", handler.StorePostHandler)
				r.Delete("/", handler.StoreDeleteHandler)
				r.Post("/restore", handler.RestoreHandler)
			})
		})
	})

	// r.Use(middleware.CreateConsentMiddleware(spaHandler))
	// r.Use(spotAuthMiddleware)

	// Provide the webapp following the SPA pattern: all non-API routes not being able
	// to be resolved within the assets directory will return the webapp entry point.
	// We wrap the SPA handler up in the Spotify Authentication middleware, which itself is wrapped inside
	// the consent middleware.
	consentMiddleware := middleware.CreateConsentMiddleware(spaHandler)
	chain := consentMiddleware(spotAuthMiddleware(spaHandler))
	r.NotFound(chain.ServeHTTP)

	return r
}

func attachSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, constants.SessionCookieName)
		if err != nil {
			// This should not never happen except some client tampers with his session.
			// But then in should fail in the spotifyAuth middleware already...
			log.Panic().Err(err).Msg("Could not access session storage!")
		}

		newCtx := context.WithValue(r.Context(), constants.FieldSession, session)

		next.ServeHTTP(w, r.WithContext(newCtx))
	})
}

func attachUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session := ctx.Value(constants.FieldSession).(*sessions.Session)

		rawUser, exists := session.Values["user"]
		if !exists {
			log.Debug().Msg("'user' not yet set in session. Going to do this.")

			// Once per session-lifetime we have to get the user ID from the spotifyClient.
			// We then cache it in the session.
			spotifyClient, err := spotifyClientFromSession(session)
			if err != nil {
				http.Error(w, err.Error(), http.StatusForbidden)
				return
			}

			rawUser, err = spotifyClient.CurrentUser()
			if err != nil {
				log.Panic().Err(err).Msg("Could not fetch information on user from Spotify!")
			}

			session.Values["user"] = rawUser

			session.Save(r, w)
		}

		user, ok := rawUser.(*spotifyAPI.PrivateUser)
		if !ok {
			// This should never happen
			log.Panic().Msg("Could not read current user from session!")
		}

		newCtx := context.WithValue(ctx, constants.FieldUser, user)

		next.ServeHTTP(w, r.WithContext(newCtx))
	})
}

func attachSpotifyClient(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session := ctx.Value(constants.FieldSession).(*sessions.Session)

		client, err := spotifyClientFromSession(session)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		newCtx := context.WithValue(ctx, constants.FieldSpotifyClient, client)

		next.ServeHTTP(w, r.WithContext(newCtx))
	})
}

func spotifyClientFromSession(session *sessions.Session) (spotify.SpotClient, error) {
	rawToken := session.Values["spotify-oauth-token"]

	tok, ok := rawToken.(*oauth2.Token)
	if !ok {
		// This happens in case a user requests the /api routes without being signed in via Spotify
		err := errors.New("Could not read Spotify token from session. User probably did not log in.")
		log.Error().Err(err).Msg("")
		return nil, err
	}

	// If auth is the mock implementation we also want to create a mock client
	return createSpotClient(tok), nil
}

func attachDAO(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		newCtx := context.WithValue(r.Context(), constants.FieldDao, dao)

		next.ServeHTTP(w, r.WithContext(newCtx))
	})
}

func attachSlot(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slot, err := checkSlotParameter(r)
		if err != nil {
			log.Debug().Err(err).Msg("Could not retrieve slot from request.")
			http.Error(w, fmt.Sprintf("Could not process request. Please make sure the given slot is valid: %s", err), http.StatusBadRequest)
			return
		}

		newCtx := context.WithValue(r.Context(), constants.FieldSlot, slot)

		next.ServeHTTP(w, r.WithContext(newCtx))
	})
}

func checkSlotParameter(r *http.Request) (int, error) {
	var slotStr = chi.URLParam(r, "slot")

	if slotStr == "" {
		return -1, errors.New("query parameter 'slot' not found")
	}

	var slot, err = strconv.Atoi(slotStr)
	if err != nil {
		return -1, errors.New("query parameter 'slot' is not a valid integer")
	}
	if slot < 0 {
		return -1, errors.New("query parameter 'slot' has to be >= 0")
	}

	return slot, nil
}

func debugLogger(nxt http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get(constants.CSRFHeaderName)

		log.Debug().Str("csrfToken", token).Stringer("url", r.URL).Interface("cookies", r.Cookies()).Msg("")

		nxt.ServeHTTP(w, r)
	})
}

type csrfErrorHandler struct{}

func (csrfErrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	failureReason := csrf.FailureReason(r)
	csrfToken := r.Header.Get(constants.CSRFHeaderName)
	csrfCookie, err := r.Cookie(constants.CSRFCookieName)
	csrfCookieContent := "ERROR: cookie not present."
	if err == nil {
		csrfCookieContent = csrfCookie.Value
	}

	http.Error(w,
		fmt.Sprintf("Failed verifying CSRF token. Supplied token: '%s'; Error: '%s'. Expect token to be contained in header '%s'. Cookie named '%s' contains '%s'",
			csrfToken, failureReason, constants.CSRFHeaderName, constants.CSRFCookieName, csrfCookieContent),
		http.StatusUnauthorized)
}
