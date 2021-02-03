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

	// names of envs
	EnvENV                 = "CASSETTE_ENV"
	EnvNetworkInterface    = "CASSETTE_NETWORK_INTERFACE"
	EnvPort                = "CASSETTE_PORT"
	EnvAppURL              = "CASSETTE_APP_URL"
	EnvSecret              = "CASSETTE_SECRET"
	EnvMongoURI            = "CASSETTE_MONGODB_URI"
	EnvSpotifyClientID     = "CASSETTE_SPOTIFY_CLIENT_ID"
	EnvSpotifyClientSecret = "CASSETTE_SPOTIFY_CLIENT_KEY"

	// keys for context fields
	FieldSession       = ctxKey("session")
	FieldDao           = ctxKey("dao")
	FieldSlot          = ctxKey("slot")
	FieldUser          = ctxKey("user")
	FieldSpotifyClient = ctxKey("spotifyClient")
)

type ctxKey string
