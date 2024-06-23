package constants

const (
	SessionCookieName       = "cassette_session"
	CSRFHeaderName          = "X-Cassette-CSRF"
	CSRFCookieName          = "cassette_csrf"
	ConsentCookieName       = "cassette_consent"
	ConsentNoticeHeaderName = "X-Cassette-Consent-Notice"
	DefaultNetworkInterface = "localhost"
	DefaultPort             = "8080"
	DefaultInternalPort     = "8081"
	JumpBackNSeconds        = 10
	WebStaticContentPath    = "./web/dist"
	OAuthCallbackRoute      = "/spotify-oauth-callback"

	// Names of envs
	EnvENV                      = "CASSETTE_ENV"
	EnvNetworkInterface         = "CASSETTE_NETWORK_INTERFACE"
	EnvInternalNetworkInterface = "CASSETTE_INTERNAL_NETWORK_INTERFACE"
	EnvPort                     = "CASSETTE_PORT"
	EnvInternalPort             = "CASSETTE_INTERNAL_PORT"
	EnvAppURL                   = "CASSETTE_APP_URL"
	EnvSecret                   = "CASSETTE_SECRET"
	EnvMongoURI                 = "CASSETTE_MONGODB_URI"
	EnvSpotifyClientID          = "CASSETTE_SPOTIFY_CLIENT_ID"
	EnvSpotifyClientSecret      = "CASSETTE_SPOTIFY_CLIENT_KEY"

	// Keys for context fields
	FieldKeySession = ctxKey(iota)
	FieldKeyDao
	FieldKeySlot
	FieldKeyUser
	FieldKeySpotifyClient

	// Keys for session values, as these are stored in the session cookie use something small
	SessionKeyUser = sessionKey(iota)
	SessionKeySpotifyToken
	SessionKeyInitiallyRequestedRoute
	SessionKeyOAuthRandomState
)

type ctxKey int
type sessionKey int
