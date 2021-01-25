package middleware

import (
	"log"
	"net/http"
	"net/url"

	"github.com/gorilla/sessions"
	spotifyAPI "github.com/zmb3/spotify"

	constants "github.com/florianloch/cassette/internal"
	"github.com/florianloch/cassette/internal/handler"
	"github.com/florianloch/cassette/internal/util"
)

// TODO: Rewrite logging and error handling
func CreateSpotifyAuthMiddleware(store *sessions.CookieStore, auth *spotifyAPI.Authenticator, redirectURL *url.URL) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, _ := store.Get(r, constants.SessionCookieName)

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

				// if isDevMode {
				log.Println("No oauth token yet...")
				// }

				// the clients browser calls this route being redirected from Spotify's authentication system
				if r.URL.Path == redirectURL.Path {
					// if isDevMode {
					log.Println("Callback route called, processing...")
					// }

					tok, err := auth.Token(randomState, r)
					if err != nil {
						http.Error(w, "Could not get auth token for Spotify", http.StatusForbidden)
						// if isDevMode {
						log.Fatal("Could not get auth token for Spotify", err)
						// }
						return
					}

					if st := r.FormValue("state"); st != randomState {
						http.NotFound(w, r)
						// if isDevMode {
						log.Fatalf("State mismatch: %s != %s\n", st, randomState)
						// }
					}

					var client = handler.SpotifyClientFromToken(tok, auth)

					currentUser, err := client.CurrentUser()
					if err != nil {
						http.Error(w, "Could not fetch information on user from Spotify", http.StatusInternalServerError)
						// if isDevMode {
						log.Fatal("Could not fetch information on current user!", err)
						// }
						// return
					}

					// if isDevMode {
					log.Println("ID of current user:", currentUser.ID)
					// }

					session.Values["user"] = currentUser
					session.Values["spotify-oauth-token"] = tok

					// redirect to the route initially requested
					var initiallyRequestedRoute string
					if initiallyRequestedRoute, ok = session.Values["initially-requested-route"].(string); !ok {
						// client should really not be here... this happens when requesting this route straight away not being
						// redirecting via Spotify. Or in case the server's session store secret changes between two requests which should not
						// be the case...
						http.Error(w, "This route should not be requested directly.", http.StatusBadRequest)
						// if isDevMode {
						log.Fatal("Client requested the callback route directly", err)
						// }
						return
					}
					// if isDevMode {
					log.Printf("Client initially requested route '%s'", initiallyRequestedRoute)
					// }

					session.Save(r, w)
					http.Redirect(w, r, initiallyRequestedRoute, http.StatusTemporaryRedirect)
				} else {
					// no token and not the callback route, we have to redirect the client to Spotify's authentification service
					var redirectTo = auth.AuthURL(randomState)
					// if isDevMode {
					log.Printf("Redirecting to Spotify's authentication service: %s", redirectTo)
					// }

					session.Values["initially-requested-route"] = r.URL.Path

					session.Save(r, w)
					http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
				}
			} else {
				next.ServeHTTP(w, r)
			}
		})
	}
}
