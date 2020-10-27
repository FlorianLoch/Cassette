package main

import (
	"encoding/gob"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"os"

	"encoding/json"

	"github.com/gorilla/context"
	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"
)

const (
	clientIDEnvName        = "spotify_client_id"
	secretKeyEnvName       = "spotify_secret_key"
	authState              = "oauth_initiated"
	sessionKeyForToken     = "spotify-oauth-token"
	webUIStaticContentPath = "/webui"
	jumpBackNSeconds       = 10
	deviceIDQueryName      = "deviceID"
)

var (
	redirectURL      *url.URL
	auth             spotify.Authenticator
	store            = sessions.NewCookieStore([]byte("something-very-secret")) // TODO Replace this by something better stored in an env
	port             = "8080"
	networkInterface = "localhost"
	appURL           = "http://localhost:8080/"
	CSRF             = csrf.Protect(
		[]byte("a-32-byte-long-key-goes-here"),
		csrf.RequestHeader("X-CSRF-Token"),
		csrf.FieldName("CSRF_token"),
		csrf.Secure(false), // TODO Change this in Prod
		// csrf.ErrorHandler(http.HandlerFunc(hterverError(403))),
	)
)

func spotifyAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session")

		// if there is no oauth token yet...
		if _, ok := session.Values[sessionKeyForToken]; !ok {
			log.Println("No oauth token yet...")

			// is this handler perhaps triggered by the client's browser calling the callback route?
			if r.URL.Path == redirectURL.Path {
				log.Println("Callback route called, processing...")

				tok, err := auth.Token(authState, r)
				if err != nil {
					http.Error(w, "Couldn't get token", http.StatusForbidden)
					log.Fatal(err)
				}
				if st := r.FormValue("state"); st != authState {
					http.NotFound(w, r)
					log.Fatalf("State mismatch: %s != %s\n", st, authState)
				}

				var client = _getSpotifyClientForRequest(tok)

				currentUser, err := client.CurrentUser()

				if err != nil {
					log.Fatal("Could not fetch information on current user!", err)
				}

				log.Println("ID of current user:", currentUser.ID)

				session.Values["user"] = currentUser
				session.Values[sessionKeyForToken] = tok

				// redirect to the route initially requested
				var initiallyRequestedRoute = session.Values["initially-requested-route"].(string)
				log.Println("Client initially requested route '" + initiallyRequestedRoute + "'!")

				session.Save(r, w)
				http.Redirect(w, r, initiallyRequestedRoute, http.StatusTemporaryRedirect)
			} else {
				// no token and not the callback route, we have to redirect the client to Spotify's authentification service
				url := auth.AuthURL(authState)
				log.Println("Redirecting to Spotify's authentication service: " + url)

				session.Values["initially-requested-route"] = r.URL.Path

				session.Save(r, w)
				http.Redirect(w, r, url, http.StatusTemporaryRedirect)
			}
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

func getSpotifyClientForRequest(r *http.Request) *spotify.Client {
	session, _ := store.Get(r, "session")

	rawToken := session.Values[sessionKeyForToken]

	return _getSpotifyClientForRequest(rawToken)
}

func getCurrentUser(r *http.Request) *spotify.PrivateUser {
	session, _ := store.Get(r, "session")

	rawUser := session.Values["user"]

	var user = &spotify.PrivateUser{}
	var ok = true
	if user, ok = rawUser.(*spotify.PrivateUser); !ok {
		// type-assert failed
		log.Fatal("Could not type-assert the stored user!")
	}

	return user
}

func _getSpotifyClientForRequest(rawToken interface{}) *spotify.Client {
	var tok = &oauth2.Token{}
	var ok = true
	if tok, ok = rawToken.(*oauth2.Token); !ok {
		// type-assert failed
		log.Fatal("Could not type-assert the stored token!")
	}

	client := auth.NewClient(tok)

	return &client
}

type m map[string]interface{}

func main() {
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}

	if os.Getenv("APP_URL") != "" {
		appURL = os.Getenv("APP_URL")
	}

	if os.Getenv("NETWORK_INTERFACE") != "" {
		networkInterface = os.Getenv("NETWORK_INTERFACE")
	}

	redirectURL, _ = url.Parse(appURL + "spotify-oauth-callback")
	auth = spotify.NewAuthenticator(redirectURL.String(), spotify.ScopeUserReadCurrentlyPlaying, spotify.ScopeUserReadPlaybackState, spotify.ScopeUserModifyPlaybackState)

	interfacePort := networkInterface + ":" + port

	gob.Register(&spotify.PrivateUser{})
	gob.Register(&oauth2.Token{})
	gob.Register(&m{})

	var cwd, _ = os.Getwd()
	var staticAssetsPath = cwd + webUIStaticContentPath

	log.Println(staticAssetsPath)

	var clientID = strings.TrimSpace(os.Getenv(clientIDEnvName))
	var clientSecret = strings.TrimSpace(os.Getenv(secretKeyEnvName))

	log.Printf("Credentials to be used authenticating with Spotify:\n\tClient ID: %s\n\tClient secret: %s\n", clientID, clientSecret)

	auth.SetAuthInfo(clientID, clientSecret)

	router := mux.NewRouter()

	router.HandleFunc("/csrfToken", csrfHandler).Methods("HEAD")

	router.Use(func (nxt http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("%s \"%s\" (%v)", r.Method, r.URL.Path, r.Header)
			nxt.ServeHTTP(w, r)
		})
	})
	router.Use(spotifyAuthMiddleware)
	// this route simply needs to be registered so that the catch all handler is able to get it?!
	router.HandleFunc("/spotify-oauth-callback", func(w http.ResponseWriter, r *http.Request) {})

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

	log.Println("Webserver started on", interfacePort)

	log.Fatal(http.ListenAndServe(interfacePort, CSRF(context.ClearHandler(http.DefaultServeMux))))
}

func csrfHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-CSRF-Token", csrf.Token(r))

	w.WriteHeader(http.StatusOK)
}

func activeDevicesHandler(w http.ResponseWriter, r *http.Request) {
	json, err := getActiveSpotifyDevices(getSpotifyClientForRequest(r))

	if err != nil {
		log.Println("Could not fetch list of active devices:", err)
		http.Error(w, "Could not fetch list of active devices from Spotify!", http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
	w.Write(json)
}

func storeHandler(w http.ResponseWriter, r *http.Request, slot int) {
	err := storeCurrentPlayerState(getSpotifyClientForRequest(r), &getCurrentUser(r).ID, slot)

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

	var deviceID = r.URL.Query().Get(deviceIDQueryName)

	if err != nil {
		http.Error(w, "Could not process request: "+err.Error(), http.StatusBadRequest)
		return
	}

	err = restorePlayerState(getSpotifyClientForRequest(r), &getCurrentUser(r).ID, slot, deviceID)

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
