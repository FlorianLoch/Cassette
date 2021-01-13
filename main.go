package main

import (
	"encoding/gob"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/florianloch/spotistate/persistence"
	"github.com/florianloch/spotistate/util"

	"github.com/gorilla/context"
	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"
)

const (
	webUIStaticContentPath  = "/webui"
	jumpBackNSeconds        = 10
	defaultNetworkInterface = "localhost"
	defaultPort             = "8080"
)

var (
	redirectURL     *url.URL
	auth            spotify.Authenticator
	store           *sessions.CookieStore
	playerStatesDAO *persistence.PlayerStatesDAO
	isDevMode       bool
)

type m map[string]interface{}

func main() {
	isDevMode = os.Getenv("ENV") == "DEV"
	log.Printf("Running in DEV mode: %t", isDevMode)

	var networkInterface = defaultNetworkInterface
	if os.Getenv("NETWORK_INTERFACE") != "" {
		networkInterface = os.Getenv("NETWORK_INTERFACE")
	}

	var port = defaultPort
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}

	var appURL = "http://" + networkInterface + ":" + port + "/"
	if os.Getenv("APP_URL") != "" {
		appURL = os.Getenv("APP_URL")
	}

	var secret32Bytes, err = util.Make32ByteSecret(os.Getenv("SECRET"))
	if err != nil {
		log.Fatal("Could not generate secret. Aborting.", err)
	}

	log.Printf("%x", secret32Bytes)

	if os.Getenv("MONGO_DB_URI") != "" {
		playerStatesDAO = persistence.NewPlayerStatesDAOFromConnectionString(os.Getenv("MONGO_DB_URI"))

	} else {
		log.Fatal("No URI for connecting to MongoDB given. Aborting.")
	}

	store = sessions.NewCookieStore(secret32Bytes)

	redirectURL, _ = url.Parse(appURL + "spotify-oauth-callback")
	auth = spotify.NewAuthenticator(redirectURL.String(), spotify.ScopeUserReadCurrentlyPlaying, spotify.ScopeUserReadPlaybackState, spotify.ScopeUserModifyPlaybackState)

	sessions.NewCookieStore([]byte("something-very-secret"))

	gob.Register(&spotify.PrivateUser{})
	gob.Register(&oauth2.Token{})
	gob.Register(&m{})

	var clientID = strings.TrimSpace(os.Getenv("SPOTIFY_CLIENT_ID"))
	var clientSecret = strings.TrimSpace(os.Getenv("SPOTIFY_CLIENT_KEY"))

	log.Printf("Credentials to be used authenticating with Spotify:\n\tClient ID: %s\n\tClient secret: %s\n", clientID, clientSecret)

	auth.SetAuthInfo(clientID, clientSecret)

	var CSRF = csrf.Protect(
		secret32Bytes,
		csrf.RequestHeader("X-CSRF-Token"),
		csrf.Secure(!isDevMode),
	)

	router := mux.NewRouter()

	router.Use(func(nxt http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("%s \"%s\" (%v)", r.Method, r.URL.Path, r.Header)
			nxt.ServeHTTP(w, r)
		})
	})

	router.Use(spotifyAuthMiddleware)
	// this route simply needs to be registered so that the catch-all-handler is able to get it?!
	router.HandleFunc("/spotify-oauth-callback", func(w http.ResponseWriter, r *http.Request) {})

	router.HandleFunc("/csrfToken", csrfHandler).Methods("HEAD")

	var cwd, _ = os.Getwd()
	var staticAssetsPath = cwd + webUIStaticContentPath
	log.Printf("Loading assets from: %s", staticAssetsPath)
	router.PathPrefix("/webui").Handler(http.StripPrefix("/webui/", http.FileServer(http.Dir(staticAssetsPath))))

	router.HandleFunc("/activeDevices", activeDevicesHandler).Methods("GET")

	router.HandleFunc("/playerStates", func(w http.ResponseWriter, r *http.Request) {
		storeHandler(w, r, -1)
	}).Methods("POST")

	router.HandleFunc("/playerStates", storeGetHandler).Methods("GET")

	router.HandleFunc("/playerStates/{slot}", func(w http.ResponseWriter, r *http.Request) {
		slot, err := checkSlotParameter(r)

		if err != nil {
			http.Error(w, "Could not process request: "+err.Error(), http.StatusBadRequest)
			return
		}

		storeHandler(w, r, slot)
	}).Methods("PUT")

	router.HandleFunc("/playerStates/{slot}", storeDeleteHandler).Methods("DELETE")

	router.HandleFunc("/playerStates/{slot}/restore", restoreHandler).Methods("POST")

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/webui/", http.StatusTemporaryRedirect)
	})

	http.Handle("/", router)

	var interfacePort = networkInterface + ":" + port
	log.Println("Webserver started on", interfacePort)
	log.Fatal(http.ListenAndServe(interfacePort, CSRF(context.ClearHandler(http.DefaultServeMux))))
}

func spotifyAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session")

		// if there is no oauth token yet...
		if _, ok := session.Values["spotify-oauth-token"]; !ok {
			// this state is used during oauth negotiation in order to prevent CSRF
			var randomState string
			if randomState, ok := session.Values["oauth-random-csrf-state"]; !ok {
				randomSecret, err := util.Make32ByteSecret("")
				if err != nil {
					log.Fatal("Failed generating a random secret for OAuth negotiation", err)
				}
				randomState = string(randomSecret)

				session.Values["oauth-random-csrf-state"] = randomState
			}

			if isDevMode {
				log.Println("No oauth token yet...")
			}

			// the clients browser calls this route being redirected from Spotify's authentication system
			if r.URL.Path == redirectURL.Path {
				if isDevMode {
					log.Println("Callback route called, processing...")
				}

				tok, err := auth.Token(randomState, r)
				if err != nil {
					http.Error(w, "Could not get auth token for Spotify", http.StatusForbidden)
					if isDevMode {
						log.Fatal("Could not get auth token for Spotify", err)
					}
					return
				}

				if st := r.FormValue("state"); st != randomState {
					http.NotFound(w, r)
					if isDevMode {
						log.Fatalf("State mismatch: %s != %s\n", st, randomState)
					}
				}

				var client = getSpotifyClientFromToken(tok)

				currentUser, err := client.CurrentUser()
				if err != nil {
					http.Error(w, "Could not fetch information on user from Spotify", http.StatusInternalServerError)
					if isDevMode {
						log.Fatal("Could not fetch information on current user!", err)
					}
					return
				}

				if isDevMode {
					log.Println("ID of current user:", currentUser.ID)
				}

				session.Values["user"] = currentUser
				session.Values["spotify-oauth-token"] = tok

				// redirect to the route initially requested
				var initiallyRequestedRoute string
				if initiallyRequestedRoute, ok = session.Values["initially-requested-route"].(string); !ok {
					// client should really not be here... this happens when requesting this route straight away not being
					// redirecting via Spotify. Or in case the server's session store secret changes between two requests which should not
					// be the case...
					http.Error(w, "This route should not be requested directly.", http.StatusBadRequest)
					if isDevMode {
						log.Fatal("Client requested the callback route directly", err)
					}
					return
				}
				if isDevMode {
					log.Printf("Client initially requested route '%s'", initiallyRequestedRoute)
				}

				session.Save(r, w)
				http.Redirect(w, r, initiallyRequestedRoute, http.StatusTemporaryRedirect)
			} else {
				// no token and not the callback route, we have to redirect the client to Spotify's authentification service
				var redirectTo = auth.AuthURL(randomState)
				if isDevMode {
					log.Printf("Redirecting to Spotify's authentication service: %s", redirectTo)
				}

				session.Values["initially-requested-route"] = r.URL.Path

				session.Save(r, w)
				http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
			}
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

func getSpotifyClientFromRequest(r *http.Request) *spotify.Client {
	session, _ := store.Get(r, "session")

	rawToken := session.Values["spotify-oauth-token"]

	return getSpotifyClientFromToken(rawToken)
}

func getCurrentUser(r *http.Request) *spotify.PrivateUser {
	session, _ := store.Get(r, "session")

	rawUser := session.Values["user"]

	var user = &spotify.PrivateUser{}
	var ok = true
	if user, ok = rawUser.(*spotify.PrivateUser); !ok {
		log.Fatal("Could not type-assert the stored user!")
	}

	return user
}

func getSpotifyClientFromToken(rawToken interface{}) *spotify.Client {
	var tok = &oauth2.Token{}
	var ok = true
	if tok, ok = rawToken.(*oauth2.Token); !ok {
		log.Fatal("Could not type-assert the stored token!")
	}

	client := auth.NewClient(tok)

	return &client
}

func csrfHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-CSRF-Token", csrf.Token(r))

	w.WriteHeader(http.StatusOK)
}

func activeDevicesHandler(w http.ResponseWriter, r *http.Request) {
	json, err := getActiveSpotifyDevices(getSpotifyClientFromRequest(r))

	if err != nil {
		log.Println("Could not fetch list of active devices:", err)
		http.Error(w, "Could not fetch list of active devices from Spotify!", http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
	w.Write(json)
}

func storeHandler(w http.ResponseWriter, r *http.Request, slot int) {
	err := storeCurrentPlayerState(getSpotifyClientFromRequest(r), &getCurrentUser(r).ID, slot)

	if err != nil {
		log.Println("Could not process request:", err)
		http.Error(w, "Could not process request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func storeGetHandler(w http.ResponseWriter, r *http.Request) {
	var playerStates = playerStatesDAO.LoadPlayerStates(getCurrentUser(r).ID)

	var json, err = json.Marshal(playerStates)

	if err != nil {
		log.Println("Could not serialize playerStates to JSON:", err)
		http.Error(w, "Could not provide player states as JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

func storeDeleteHandler(w http.ResponseWriter, r *http.Request) {
	var slot, err = checkSlotParameter(r)

	if err != nil {
		log.Println("Could not process request:", err)
		http.Error(w, "Could not process request: "+err.Error(), http.StatusBadRequest)
		return
	}

	var playerStates = playerStatesDAO.LoadPlayerStates(getCurrentUser(r).ID)

	if slot >= len(playerStates.States) || slot < 0 {
		http.Error(w, "Could not process request: 'slot' is not in the range of exisiting slots", http.StatusInternalServerError)
		return
	}

	playerStates.States = append(playerStates.States[:slot], playerStates.States[slot+1:]...)

	playerStatesDAO.SavePlayerStates(playerStates)
}

func restoreHandler(w http.ResponseWriter, r *http.Request) {
	var slot, err = checkSlotParameter(r)

	var deviceID = r.URL.Query().Get("deviceID")

	if err != nil {
		http.Error(w, "Could not process request: "+err.Error(), http.StatusBadRequest)
		return
	}

	err = restorePlayerState(getSpotifyClientFromRequest(r), &getCurrentUser(r).ID, slot, deviceID)

	if err != nil {
		http.Error(w, "Could not process request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func checkSlotParameter(r *http.Request) (int, error) {
	var rawSlot, ok = mux.Vars(r)["slot"]

	if !ok {
		return -1, errors.New("query parameter 'slot' not found or more than one provided")
	}

	var slot, err = strconv.Atoi(rawSlot)

	if err != nil {
		return -1, errors.New("query parameter 'slot' is not a valid integer")
	}
	if slot < 0 {
		return -1, errors.New("query parameter 'slot' has to be >= 0")
	}

	return slot, nil
}
