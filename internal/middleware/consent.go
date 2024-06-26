package middleware

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/hlog"

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

			// To make testing easier, we send the cookie back once we received it.
			// By this, the client will be able to put the cookie in its jar.
			// For browsers, it should make no difference as the user already created the cookie.
			// But we need to set the path explicitly, otherwise multiple cookies will be set depending on the
			// request route.
			// By this, the cookie also never invalidates (assuming this route gets requested once in its lifetime).
			// ATTENTION: As the cookie is retrieved from the request, it does not contain information like max-age.
			// Therefore, this has to be set again, otherwise max-age will be set to 0 and therefore override
			// the max-age of the cookie being present in the browser already.
			cookie.Path = "/"
			cookie.MaxAge = 10 * 60 * 60 * 24 * 365 // 10 years, keep this in sync with consent middleware in the frontend

			http.SetCookie(w, cookie)

			next.ServeHTTP(w, r)
		})
	}
}

func consentGiven(r *http.Request) (*http.Cookie, bool) {
	cookie, err := r.Cookie(constants.ConsentCookieName)
	if errors.Is(err, http.ErrNoCookie) {
		hlog.FromRequest(r).Debug().Msg("User did not yet give her/his consent.")

		return nil, false
	}

	ts, err := strconv.ParseInt(cookie.Value, 10, 64)
	if err != nil {
		hlog.FromRequest(r).Debug().Msg("Consent cookie does not contain valid timestamp.")

		return nil, false
	}

	date := time.Unix(ts, 0).UTC()
	hlog.FromRequest(r).Debug().Msgf("User already gave her/his consent at '%s'.", date)

	return cookie, true
}
