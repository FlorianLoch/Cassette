package middleware

import (
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

// CreateConsentMiddleware returns a middleware ensuring that as long as an user cannot provide a valid
// consent cookie she/he will only be served the the SPA. As no other route will be served no cookie etc. will
// be set. All the user can do is requesting the main SPA - but it won't work and no data will be
// processed, stored or handled in any other way.
func CreateConsentMiddleware(spaHandler http.Handler, consentCookieName string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var cookie, err = r.Cookie(consentCookieName)
			if err == http.ErrNoCookie {
				log.Debug().Msg("User did not yet give her/his consent. Serving the consent page.")

				spaHandler.ServeHTTP(w, r)

				return
			}

			cookieValue, _ := url.QueryUnescape(cookie.Value)
			log.Debug().Msgf("User already gave her/his consent at '%s'.", cookieValue)

			next.ServeHTTP(w, r)
		})
	}
}
