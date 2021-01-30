package middleware

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/gorilla/sessions"
	spotifyAPI "github.com/zmb3/spotify"

	"github.com/florianloch/cassette/internal/util"
	"github.com/rs/zerolog/log"
)

func CreateSpotifyAuthMiddleware(
	store *sessions.CookieStore,
	auth *spotifyAPI.Authenticator,
	redirectURL *url.URL) (func(http.Handler) http.Handler, http.HandlerFunc) {
	spotAuthMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session := r.Context().Value("session").(*sessions.Session)

			if _, ok := session.Values["spotify-oauth-token"]; ok {
				log.Debug().Msg("OAuth token already present. Nothing to do.")

				next.ServeHTTP(w, r)

				return
			}

			if r.URL.Path == redirectURL.Path {
				// Let the callback route resp. its handler handle this case
				next.ServeHTTP(w, r)
				return
			}

			log.Debug().Msg("No OAuth token yet. Initializing OAuth flow...")

			randomState := randomStateFromSession(session)

			// No token yet and not the callback route, we have to redirect the client to Spotify's
			// authentification service
			redirectTo := auth.AuthURL(randomState)
			log.Debug().Str("authURL", redirectTo).Msg("Redirecting to Spotify's auth service.")

			// Store the currently requested route in order to be able to forward the user after successful
			// OAuth flow
			session.Values["initially-requested-route"] = r.URL.Path
			session.Save(r, w)

			http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
		})
	}
	spotOAuthCBHandler := func(w http.ResponseWriter, r *http.Request) {
		session := r.Context().Value("session").(*sessions.Session)

		randomState := randomStateFromSession(session)

		token, err := auth.Token(randomState, r)
		if err != nil {
			http.Error(w, "Could not get auth token for Spotify", http.StatusForbidden)
			log.Error().Err(err).Msg("Could not get auth token for Spotify.")
			return
		}

		if state := r.FormValue("state"); state != randomState {
			http.NotFound(w, r)
			log.Error().Str("stateGiven", state).Str("stateExpected", randomState).Msg("State mismatch in OAuth callback.")
			return
		}

		session.Values["spotify-oauth-token"] = token

		// Redirect to the route initially requested
		initiallyRequestedRoute, ok := session.Values["initially-requested-route"]
		if !ok {
			// client should really not be here... this happens when requesting this route straight away not being
			// redirecting via Spotify. Or in case the session got lost with should not occur.
			http.Error(w, "This route should not be requested directly.", http.StatusForbidden)
			log.Error().Msg("Client requested the OAuth callback route directly.")
			return
		}

		session.Save(r, w)

		http.Redirect(w, r, initiallyRequestedRoute.(string), http.StatusTemporaryRedirect)
	}

	return spotAuthMiddleware, spotOAuthCBHandler
}

func randomStateFromSession(session *sessions.Session) string {
	// This state is used during oauth negotiation in order to prevent CSRF
	var randomState string
	if _, ok := session.Values["oauth-random-state"]; !ok {
		randomSecret, err := util.Make32ByteSecret("") // returns a random secret
		if err != nil {
			log.Panic().Err(err).Msg("Failed to generate a random secret for OAuth negotiation.")
		}

		session.Values["oauth-random-state"] = fmt.Sprintf("%x", randomSecret)
	}
	randomState = (session.Values["oauth-random-state"]).(string)

	return randomState
}
