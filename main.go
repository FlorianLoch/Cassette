package main

import (
	"encoding/gob"
	"net/http"
	"net/url"
	"os"

	constants "github.com/florianloch/cassette/internal"
	"github.com/florianloch/cassette/internal/handler"
	"github.com/florianloch/cassette/internal/middleware"
	"github.com/florianloch/cassette/internal/persistence"
	"github.com/florianloch/cassette/internal/util"

	"github.com/gorilla/context"
	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"
)

var (
	redirectURL          *url.URL
	auth                 *spotify.Authenticator
	store                *sessions.CookieStore
	playerStatesDAO      *persistence.PlayerStatesDAO
	isDevMode            bool
	webStaticContentPath = "/web/dist"
)

type m map[string]interface{}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	isDevMode = util.Env(constants.EnvENV, "") == "DEV"
	if isDevMode {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("Running in DEV mode. Being more verbose. Set environment variable 'ENV' to 'DEV' to activate.")
	}

	var networkInterface = util.Env(constants.EnvNetworkInterface, constants.DefaultNetworkInterface)
	// we also have to check for "PORT" as that is how Heroku tells the app where to listen
	var port = util.Env(constants.EnvPort, os.Getenv("PORT"))
	var appURL = util.Env(constants.EnvAppURL, "http://"+networkInterface+":"+port+"/")

	var secret32Bytes, err = util.Make32ByteSecret(util.Env(constants.EnvSecret, ""))
	if err != nil {
		log.Fatal().Err(err).Msg("Could not generate secret. Aborting.")
	}

	var mongoDBURI = util.Env(constants.EnvMongoURI, "")
	if mongoDBURI == "" {
		log.Fatal().Msg("No URI for connecting to MongoDB given. Aborting.")
	}
	dao, err := persistence.Connect(mongoDBURI)

	store = sessions.NewCookieStore(secret32Bytes)

	redirectURL, _ = url.Parse(appURL + "spotify-oauth-callback")
	tmp := spotify.NewAuthenticator(redirectURL.String(), spotify.ScopeUserReadCurrentlyPlaying, spotify.ScopeUserReadPlaybackState, spotify.ScopeUserModifyPlaybackState)
	auth = &tmp

	gob.Register(&spotify.PrivateUser{})
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
		csrf.RequestHeader(constants.CSRFTokenName),
		csrf.CookieName(constants.CSRFTokenName),
		csrf.Secure(!isDevMode),
	)

	var cwd, _ = os.Getwd()
	var staticAssetsPath = cwd + webStaticContentPath
	var spaHandler = handler.NewSpaHandler(staticAssetsPath, "index.html")

	rootRouter := mux.NewRouter()
	apiRouter := rootRouter.PathPrefix("/api").Subrouter()

	rootRouter.Use(middleware.CreateConsentMiddleware(spaHandler))
	rootRouter.Use(middleware.CreateSpotifyAuthMiddleware(store, auth, redirectURL))

	apiRouter.Use(csrfMiddleware)

	// if isDevMode {
	// 	apiRouter.Use(func(nxt http.Handler) http.Handler {
	// 		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 			log.Printf("%s \"%s\"", r.Method, r.URL.Path)
	// 			spew.Print(r.Header)
	// 			nxt.ServeHTTP(w, r)
	// 		})
	// 	})
	// }

	// this route simply needs to be registered so that the middleware registered at the router gets invoked
	// on requests for it
	apiRouter.HandleFunc("/spotify-oauth-callback", func(w http.ResponseWriter, r *http.Request) {})

	apiRouter.HandleFunc("/csrfToken", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(constants.CSRFTokenName, csrf.Token(r))

		w.WriteHeader(http.StatusOK)
	}).Methods("HEAD")

	apiRouter.HandleFunc("/activeDevices", func(w http.ResponseWriter, r *http.Request) {
		handler.ActiveDevicesHandler(w, r, store, auth)
	}).Methods("GET")

	apiRouter.HandleFunc("/playerStates", func(w http.ResponseWriter, r *http.Request) {
		handler.StorePostHandler(w, r, store, auth, dao, -1)
	}).Methods("POST")

	apiRouter.HandleFunc("/playerStates", func(w http.ResponseWriter, r *http.Request) {
		handler.StoreGetHandler(w, r, store, dao)
	}).Methods("GET")

	apiRouter.HandleFunc("/playerStates/{slot}", func(w http.ResponseWriter, r *http.Request) {
		slot, err := handler.CheckSlotParameter(r)
		if err != nil {
			log.Debug().Err(err).Msg("Could not process request.")
			http.Error(w, "Could not process request. Please check whether the given slot is valid.", http.StatusBadRequest)
			return
		}

		handler.StorePostHandler(w, r, store, auth, dao, slot)
	}).Methods("PUT")

	apiRouter.HandleFunc("/playerStates/{slot}", func(w http.ResponseWriter, r *http.Request) {
		handler.StoreDeleteHandler(w, r, store, dao)
	}).Methods("DELETE")

	apiRouter.HandleFunc("/playerStates/{slot}/restore", func(w http.ResponseWriter, r *http.Request) {
		handler.RestoreHandler(w, r, store, auth, dao)
	}).Methods("POST")

	apiRouter.HandleFunc("/you", func(w http.ResponseWriter, r *http.Request) {
		handler.UserExportHandler(w, r, store, dao)
	}).Methods("GET")
	apiRouter.HandleFunc("/you", func(w http.ResponseWriter, r *http.Request) {
		handler.UserDeleteHandler(w, r, store, dao)
	}).Methods("DELETE")

	// provide the webapp following the SPA pattern: all non-API routes not being able
	// to be resolved within the assets directory will return the webapp entry point.
	// ATTENTION: This is a catch-all route; every route declared after this one will not match any request!
	log.Info().Msgf("Loading assets from: '%s'", staticAssetsPath)
	rootRouter.PathPrefix("/").Handler(spaHandler)

	http.Handle("/", rootRouter)

	var interfacePort = networkInterface + ":" + port
	log.Info().Msgf("Webserver started on %s", interfacePort)

	err = http.ListenAndServe(interfacePort, context.ClearHandler(http.DefaultServeMux))
	log.Fatal().Err(err).Msg("Server terminated.")
}
