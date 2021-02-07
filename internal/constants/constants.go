package constants

const (
	SessionCookieName       = "cassette_session"
	CSRFHeaderName          = "X-Cassette-CSRF"
	CSRFCookieName          = "cassette_csrf"
	ConsentCookieName       = "cassette_consent"
	ConsentNoticeHeaderName = "X-Cassette-Consent-Notice"
	DefaultNetworkInterface = "localhost"
	DefaultPort             = "8080"
	JumpBackNSeconds        = 10
	WebStaticContentPath    = "./web/dist"
	OAuthCallbackRoute      = "/spotify-oauth-callback"

	// Names of envs
	EnvENV                 = "CASSETTE_ENV"
	EnvNetworkInterface    = "CASSETTE_NETWORK_INTERFACE"
	EnvPort                = "CASSETTE_PORT"
	EnvAppURL              = "CASSETTE_APP_URL"
	EnvSecret              = "CASSETTE_SECRET"
	EnvMongoURI            = "CASSETTE_MONGODB_URI"
	EnvSpotifyClientID     = "CASSETTE_SPOTIFY_CLIENT_ID"
	EnvSpotifyClientSecret = "CASSETTE_SPOTIFY_CLIENT_KEY"

	// Keys for context fields
	FieldKeySession       = ctxKey("session")
	FieldKeyDao           = ctxKey("dao")
	FieldKeySlot          = ctxKey("slot")
	FieldKeyUser          = ctxKey("user")
	FieldKeySpotifyClient = ctxKey("spotifyClient")

	// Keys for session values, as these are stored in the session cookie use something small
	SessionKeyUser                    = sessionKey(0)
	SessionKeySpotifyToken            = sessionKey(1)
	SessionKeyInitiallyRequestedRoute = sessionKey(2)
	SessionKeyOAuthRandomState        = sessionKey(3)
)

type ctxKey string
type sessionKey int
