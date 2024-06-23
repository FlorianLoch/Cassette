package internal

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi"
	chiMiddleware "github.com/go-chi/chi/middleware"
	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	metrics "github.com/slok/go-http-metrics/metrics/prometheus"
	httpMetricsMiddleware "github.com/slok/go-http-metrics/middleware"
	"github.com/slok/go-http-metrics/middleware/std"
	spotifyAPI "github.com/zmb3/spotify"
	"golang.org/x/oauth2"

	"github.com/florianloch/cassette/internal/constants"
	"github.com/florianloch/cassette/internal/handler"
	"github.com/florianloch/cassette/internal/middleware"
	"github.com/florianloch/cassette/internal/persistence"
	"github.com/florianloch/cassette/internal/spotify"
	"github.com/florianloch/cassette/internal/util"
)

var (
	auth  spotify.SpotAuthenticator
	store *sessions.CookieStore
	dao   persistence.PlayerStatesPersistor
	// createSpotClient is required to use different initilisation code for testing
	// and for production environment
	createSpotClient spotClientCreator
)

type spotClientCreator func(token *oauth2.Token) spotify.SpotClient
type m map[string]interface{}

func RunInProduction() {
	ctx, cancelFn := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancelFn()

	isDevMode := util.Env(constants.EnvENV, "") == "DEV"

	networkInterface := util.Env(constants.EnvNetworkInterface, constants.DefaultNetworkInterface)
	// We also have to check for "PORT" as that is how Heroku/Dokku etc. tells the app where to listen
	port := util.Env(constants.EnvPort, os.Getenv("PORT"))
	appURL := util.Env(constants.EnvAppURL, "http://"+networkInterface+":"+port+"/")

	mongoDBURI := util.Env(constants.EnvMongoURI, "")
	if mongoDBURI == "" {
		log.Fatal().Msg("No URI for connecting to MongoDB given. Aborting.")
	}
	var err error
	dao, err = persistence.Connect(mongoDBURI)
	if err != nil {
		log.Fatal().Err(err).Str("mongoDBURI", mongoDBURI).Msg("Failed connecting to MongoDB.")
	}

	redirectURL, err := url.Parse(appURL)
	if err != nil {
		log.Fatal().Err(err).Str("appURL", appURL).Msgf("'%s' variable is not set to a valid value.", constants.EnvAppURL)
	}
	redirectURL.Path = constants.OAuthCallbackRoute

	tmp := spotifyAPI.NewAuthenticator(redirectURL.String(), spotifyAPI.ScopeUserReadCurrentlyPlaying, spotifyAPI.ScopeUserReadPlaybackState, spotifyAPI.ScopeUserModifyPlaybackState)
	auth = &tmp

	clientID := util.Env(constants.EnvSpotifyClientID, "")
	clientSecret := util.Env(constants.EnvSpotifyClientSecret, "")

	if clientID == "" || clientSecret == "" {
		log.Fatal().Msgf("Please make sure '%s' and '%s' are set. Aborting.", constants.EnvSpotifyClientID, constants.EnvSpotifyClientSecret)
	}

	auth.SetAuthInfo(clientID, clientSecret)

	createSpotClient = func(token *oauth2.Token) spotify.SpotClient {
		client := auth.NewClient(token)

		return spotify.NewSpotClientWithRetry(&client, 2, 100*time.Millisecond)
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal().Err(err).Msg("Could not get current working directory.")
	}

	publicRouter := chi.NewRouter()

	internalRouter := chi.NewRouter()

	promMiddleware := httpMetricsMiddleware.New(httpMetricsMiddleware.Config{
		Recorder: metrics.NewRecorder(metrics.Config{}),
	})

	publicRouter.Use(std.HandlerProvider("", promMiddleware))

	setupAPI(cwd, isDevMode, publicRouter)

	internalRouter.Handle("/internal/metrics", promhttp.Handler())

	publicServerAddr := fmt.Sprintf("%s:%s", networkInterface, port)
	internalServerAddr := fmt.Sprintf("%s:%s",
		util.Env(constants.EnvInternalNetworkInterface, constants.DefaultNetworkInterface),
		util.Env(constants.EnvInternalPort, constants.DefaultInternalPort))

	publicServer := http.Server{
		Addr:    publicServerAddr,
		Handler: publicRouter,
	}
	internalServer := http.Server{
		Addr:    internalServerAddr,
		Handler: internalRouter,
	}

	serveWG := &sync.WaitGroup{}
	serveWG.Add(3)

	go func() {
		defer serveWG.Done()

		<-ctx.Done()

		log.Debug().Msg("Shutdown signal received. Shutting down server...")

		timeout, cancelFn := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelFn()

		if err := publicServer.Shutdown(timeout); err != nil {
			log.Error().Err(err).Msg("Failed to shutdown public server gracefully.")
		}

		timeout, cancelFn = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelFn()

		if err := internalServer.Shutdown(timeout); err != nil {
			log.Error().Err(err).Msg("Failed to shutdown internal server gracefully.")
		}
	}()

	go func() {
		defer serveWG.Done()

		if err := publicServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("Failed running public server.")
		}
	}()

	go func() {
		defer serveWG.Done()

		if err := internalServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("Failed running internal server.")
		}
	}()

	log.Info().Msgf("Public server is ready to handle requests at http://%s", publicServerAddr)
	log.Info().Msgf("Internal server is ready to handle requests at http://%s", internalServerAddr)

	<-ctx.Done()

	log.Info().Msg("Waiting for connections to be closed and for server to shutdown...")

	serveWG.Wait()

	log.Info().Msg("Server have been shut down. Bye.")
}

func SetupForTest(
	daoMock persistence.PlayerStatesPersistor,
	authMock spotify.SpotAuthenticator,
	spotClientMockCreator spotClientCreator,
	webRoot string) http.Handler {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)

	dao = daoMock

	auth = authMock

	createSpotClient = spotClientMockCreator

	r := chi.NewRouter()
	setupAPI(webRoot, true, r)

	return r
}

func setupAPI(webRoot string, isDevMode bool, r chi.Router) {
	if isDevMode {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		log.Debug().Msg("Running in PROD mode. Being less verbose. Set environment variable 'ENV' to 'DEV' to activate.")
	}

	gob.Register(&spotifyAPI.PrivateUser{})
	gob.Register(&oauth2.Token{})
	gob.Register(&m{})
	gob.Register(constants.SessionKeyUser) // could be any value, just needs to be of type session.sessionKey

	secret32Bytes, err := util.Make32ByteSecret(util.Env(constants.EnvSecret, ""))
	if err != nil {
		log.Fatal().Err(err).Msg("Could not generate secret. Aborting.")
	}

	store = sessions.NewCookieStore(secret32Bytes)
	store.Options.HttpOnly = true
	store.Options.Secure = !isDevMode
	store.Options.SameSite = http.SameSiteLaxMode

	csrfMiddleware := csrf.Protect(
		secret32Bytes,
		csrf.RequestHeader(constants.CSRFHeaderName),
		csrf.CookieName(constants.CSRFCookieName),
		csrf.Secure(!isDevMode),
		csrf.HttpOnly(true),
		csrf.SameSite(csrf.SameSiteLaxMode),
		csrf.MaxAge(60*60*24*365), // Cookie is valid for 1 year
		csrf.ErrorHandler(csrfErrorHandler{}),
	)

	spotAuthMiddleware, spotOAuthCBHandler := middleware.CreateSpotifyAuthMiddleware(auth)

	staticAssetsPath := filepath.Join(webRoot, constants.WebStaticContentPath)
	spaHandler := handler.
		NewSpaHandler(staticAssetsPath, "index.html").
		SetFileServer(http.FileServer(http.Dir(staticAssetsPath)))
	log.Info().Msgf("Loading assets from: '%s'", staticAssetsPath)

	r.Use(chiMiddleware.Compress(5))
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

	r.Get(constants.OAuthCallbackRoute, spotOAuthCBHandler)

	r.Route("/api", func(r chi.Router) {
		r.Use(csrfMiddleware)

		r.Head("/csrfToken", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(constants.CSRFHeaderName, csrf.Token(r))
			w.WriteHeader(http.StatusOK)
		})

		r.With(attachDAO).With(attachUser).Route("/you", func(r chi.Router) {
			r.Get("/", handler.UserExportHandler)
			r.Delete("/", handler.UserDeleteHandler)
		})

		r.With(attachSpotifyClient).Get("/activeDevices", handler.ActiveDevicesHandler)

		r.With(attachSpotifyClient).With(attachDAO).With(attachUser).Route("/playerStates", func(r chi.Router) {
			r.Post("/", handler.PlayerStatesPostHandler)
			r.Get("/", handler.PlayerStatesGetHandler)
			r.With(attachSlot).Route("/{slot}", func(r chi.Router) {
				r.Put("/", handler.PlayerStatesPostHandler)
				r.Delete("/", handler.PlayerStatesDeleteHandler)
				r.Post("/restore", handler.PlayerStatesRestoreHandler)
			})
		})

		r.NotFound(http.NotFound)
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
}

func attachSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, constants.SessionCookieName)
		if err != nil {
			// This should not never happen except some client tampers with his session.
			hlog.FromRequest(r).Error().Err(err).Msg("Could not access session storage!")
			http.Error(w, "Session is invalid. Please delete your session cookie and try again.", http.StatusBadRequest)
			return
		}

		newCtx := context.WithValue(r.Context(), constants.FieldKeySession, session)

		next.ServeHTTP(w, r.WithContext(newCtx))
	})
}

func attachUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session := ctx.Value(constants.FieldKeySession).(*sessions.Session)

		rawUser, exists := session.Values[constants.SessionKeyUser]
		if !exists {
			hlog.FromRequest(r).Debug().Msg("'user' not yet set in session. Going to add it.")

			// Once per session-lifetime we have to get the user ID from the spotifyClient.
			// We then cache it in the session.
			spotifyClient, err := spotifyClientFromSession(session)
			if err != nil {
				hlog.FromRequest(r).Error().Err(err).Msg("Could not initialize Spotify client for user!")
				http.Error(w, err.Error(), http.StatusForbidden)
				return
			}

			rawUser, err = spotifyClient.CurrentUser()
			if err != nil {
				hlog.FromRequest(r).Panic().Err(err).Msg("Could not fetch information on user from Spotify!")
				return
			}

			session.Values[constants.SessionKeyUser] = rawUser

			err = session.Save(r, w)
			if err != nil {
				// This should not happen. We can continue processing the request, the next call to this function
				// will try again to attach the user to the session.
				hlog.FromRequest(r).Error().Err(err).Msg("Could not update user's session.")
			}
		}

		user, ok := rawUser.(*spotifyAPI.PrivateUser)
		if !ok {
			// This should never happen
			hlog.FromRequest(r).Panic().Msg("Could not read current user from session!")
			return
		}

		newCtx := context.WithValue(ctx, constants.FieldKeyUser, user)

		next.ServeHTTP(w, r.WithContext(newCtx))
	})
}

func attachSpotifyClient(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session := ctx.Value(constants.FieldKeySession).(*sessions.Session)

		client, err := spotifyClientFromSession(session)
		if err != nil {
			hlog.FromRequest(r).Error().Err(err).Msg("Could not initialize Spotify client for user!")
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		newCtx := context.WithValue(ctx, constants.FieldKeySpotifyClient, client)

		next.ServeHTTP(w, r.WithContext(newCtx))
	})
}

func spotifyClientFromSession(session *sessions.Session) (spotify.SpotClient, error) {
	rawToken := session.Values[constants.SessionKeySpotifyToken]

	tok, ok := rawToken.(*oauth2.Token)
	if !ok {
		// This happens in case a user requests the /api routes without being signed in via Spotify
		return nil, errors.New("Could not read Spotify token from session. User probably did not log in.")
	}

	return createSpotClient(tok), nil
}

func attachDAO(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		newCtx := context.WithValue(r.Context(), constants.FieldKeyDao, dao)

		next.ServeHTTP(w, r.WithContext(newCtx))
	})
}

func attachSlot(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slot, err := checkSlotParameter(r)
		if err != nil {
			hlog.FromRequest(r).Debug().Err(err).Msg("Could not retrieve slot from request.")
			http.Error(w, fmt.Sprintf("Could not process request. Please make sure the given slot is valid: %s", err), http.StatusBadRequest)
			return
		}

		newCtx := context.WithValue(r.Context(), constants.FieldKeySlot, slot)

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
