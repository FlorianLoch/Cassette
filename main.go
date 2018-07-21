// This example demonstrates how to authenticate with Spotify.
// In order to run this example yourself, you'll need to:
//
//  1. Register an application at: https://developer.spotify.com/my-applications/
//       - Use "http://localhost:8080/callback" as the redirect URI
//  2. Set the SPOTIFY_ID environment variable to the client ID you got in step 1.
//  3. Set the SPOTIFY_SECRET environment variable to the client secret from step 1.
package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"os"

	"encoding/gob"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"
)

const (
	clientIDEnvName    = "spotify_client_id"
	secretKeyEnvName   = "spotify_secret_key"
	authState          = "oauth_initiated"
	sessionKeyForToken = "spotify-oauth-token"
	interfacePort      = "localhost:8080"
)

var store = sessions.NewCookieStore([]byte("something-very-secret"))

var (
	redirectURL, _ = url.Parse("http://localhost:8080/spotify-oauth-callback")
	auth           = spotify.NewAuthenticator(redirectURL.String(), spotify.ScopeUserReadCurrentlyPlaying, spotify.ScopeUserReadPlaybackState, spotify.ScopeUserModifyPlaybackState)
)

func spotifyAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Called Spotify Auth middleware...")
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

	tokRaw := session.Values[sessionKeyForToken]
	var tok = &oauth2.Token{}
	var ok = true
	if tok, ok = tokRaw.(*oauth2.Token); !ok {
		// type-assert failed
		log.Fatal("Could not type-assert the stored token!")
	}

	client := auth.NewClient(tok)

	return &client
}

type m map[string]interface{}

func main() {
	gob.Register(&oauth2.Token{})
	gob.Register(&m{})

	var clientID = strings.TrimSpace(os.Getenv(clientIDEnvName))
	var clientSecret = strings.TrimSpace(os.Getenv(secretKeyEnvName))

	log.Printf("Credentials to be used authenticating with Spotify:\n\tClient ID: %s\n\tClient secret: %s\n", clientID, clientSecret)

	auth.SetAuthInfo(clientID, clientSecret)

	router := mux.NewRouter()
	router.Use(spotifyAuthMiddleware)
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("This is a catch-all route"))
		log.Println("Got request for:", r.URL.String())
	})
	router.HandleFunc("/spotify-oauth-callback", func(w http.ResponseWriter, r *http.Request) {})
	router.HandleFunc("/app", func(w http.ResponseWriter, r *http.Request) {
		var client = getSpotifyClientForRequest(r)

		var playerState, err = client.PlayerState()

		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Found your %s (%s)\n", playerState.Device.Type, playerState.Device.Name)
	})

	http.Handle("/", router)

	log.Println("Webserver started on", interfacePort)

	http.ListenAndServe(interfacePort, context.ClearHandler(http.DefaultServeMux))
}
