package middleware

import (
	"fmt"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/rs/zerolog/hlog"

	"github.com/florianloch/cassette/internal/constants"
	"github.com/florianloch/cassette/internal/spotify"
	"github.com/florianloch/cassette/internal/util"
)

func CreateSpotifyAuthMiddleware(auth spotify.SpotAuthenticator) (func(http.Handler) http.Handler, http.HandlerFunc) {
	spotAuthMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session := r.Context().Value(constants.FieldKeySession).(*sessions.Session)

			if _, ok := session.Values[constants.SessionKeySpotifyToken]; ok {
				hlog.FromRequest(r).Debug().Msg("OAuth token already present. Nothing to do.")

				next.ServeHTTP(w, r)

				return
			}

			if r.URL.Path == constants.OAuthCallbackRoute {
				// Let the callback route resp. its handler handle this case
				next.ServeHTTP(w, r)
				return
			}

			hlog.FromRequest(r).Debug().Msg("No OAuth token yet. Initializing OAuth flow...")

			randomState, err := generateRandomState()
			if err != nil {
				hlog.FromRequest(r).Panic().Err(err).Msg("Failed to generate a random secret for OAuth negotiation.")
				return
			}
			session.Values[constants.SessionKeyOAuthRandomState] = randomState

			// No token yet and not the callback route, we have to redirect the client to Spotify's
			// authentification service
			redirectTo := auth.AuthURL(randomState)
			hlog.FromRequest(r).Debug().Str("authURL", redirectTo).Msg("Redirecting to Spotify's auth service.")

			// Store the currently requested route in order to be able to forward the user after successful
			// OAuth flow
			session.Values[constants.SessionKeyInitiallyRequestedRoute] = r.URL.Path
			err = session.Save(r, w)
			if err != nil {
				// Should not happen, if it does randomState can also not be saved to session - therefore callback
				// cannot be handled successfully. So fail early and let the user retry.
				hlog.FromRequest(r).Panic().Err(err).Msg("Failed to save client's session.")
				return
			}

			http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
		})
	}
	spotOAuthCBHandler := func(w http.ResponseWriter, r *http.Request) {
		session := r.Context().Value(constants.FieldKeySession).(*sessions.Session)

		rawRandomState, ok := session.Values[constants.SessionKeyOAuthRandomState]
		if !ok {
			// Might happen in case user requests the OAuth callback route after having
			// successfully initialized the session already (because of pruning randomState
			// from session after successful initialization)
			hlog.FromRequest(r).Error().Msg("Failed to retrieve randomState from session.")
			http.Error(w, "Session does not contain OAuth state", http.StatusBadRequest)
			return
		}
		randomState := rawRandomState.(string)

		if state := r.FormValue("state"); state != randomState {
			hlog.FromRequest(r).Error().
				Str("stateGiven", state).
				Str("stateExpected", randomState).
				Msg("State mismatch in OAuth callback.")
			http.Error(w, "State mismatch in OAuth callback", http.StatusBadRequest)
			return
		}

		token, err := auth.Token(randomState, r)
		if err != nil {
			hlog.FromRequest(r).Error().Err(err).Msg("Could not get auth token for Spotify.")
			http.Error(w, "Could not get auth token for Spotify", http.StatusForbidden)
			return
		}

		// Redirect to the route initially requested
		initiallyRequestedRoute, ok := session.Values[constants.SessionKeyInitiallyRequestedRoute]
		if !ok {
			// This should never happen, except in the scenario described with randomState above - but then
			// processing should already be stopped above.
			hlog.FromRequest(r).Error().Msg("Could not get initiallyRequestedRoute from session. This should never happen.")
			return
		}

		// Clean up the session, remove entries not needed any longer...
		delete(session.Values, constants.SessionKeyInitiallyRequestedRoute)
		delete(session.Values, constants.SessionKeyOAuthRandomState)

		session.Values[constants.SessionKeySpotifyToken] = token
		err = session.Save(r, w)
		if err != nil {
			hlog.FromRequest(r).Error().Err(err).Msg("Could not update user's session.")
			http.Error(w, "Could not update user's session", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, initiallyRequestedRoute.(string), http.StatusTemporaryRedirect)
	}

	return spotAuthMiddleware, spotOAuthCBHandler
}

func generateRandomState() (string, error) {
	// This state is used during OAuth negotiation in order to prevent CSRF
	randomSecret, err := util.Make32ByteSecret("") // Returns a random secret
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", randomSecret), nil
}
