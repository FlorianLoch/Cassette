package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"

	"github.com/florianloch/cassette/internal/constants"
)

// CreateConsentMiddleware returns a middleware ensuring that as long as an user cannot provide a valid
// consent cookie she/he will only be served the the SPA. As no other route will be served no cookie etc. will
// be set. All the user can do is requesting the main SPA - but it won't work and no data will be
// processed, stored or handled in any other way.
func CreateConsentMiddleware(spaHandler http.Handler) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, ok := consentGiven(r)
			if !ok {
				w.Header().Add(constants.ConsentNoticeHeaderName, "ATTENTION: consent not given yet.")
				spaHandler.ServeHTTP(w, r)

				return
			}

			// To make testing easier we send the cookie back once we received it.
			// By this the client will be able to put the cookie in its jar.
			// For browsers it should make no difference as the user already created the cookie.
			http.SetCookie(w, cookie)

			next.ServeHTTP(w, r)
		})
	}
}

func consentGiven(r *http.Request) (*http.Cookie, bool) {
	var cookie, err = r.Cookie(constants.ConsentCookieName)
	if err == http.ErrNoCookie {
		log.Debug().Msg("User did not yet give her/his consent. Serving the consent page.")

		return nil, false
	}

	ts, err := strconv.ParseInt(cookie.Value, 10, 64)
	if err != nil {
		log.Debug().Msg("Consent cookie does not contain valid timestamp. Serving the consent page.")

		return nil, false
	}

	var date = time.Unix(ts, 0).UTC()
	log.Debug().Msgf("User already gave her/his consent at '%s'.", date)

	return cookie, true
}
