package internal

const (
	SessionCookieName       = "cassette_session"
	CSRFTokenName           = "cassette_csrf_token"
	ConsentCookieName       = "cassette_consent"
	DefaultNetworkInterface = "localhost"
	DefaultPort             = "8080"
	JumpBackNSeconds        = 10

	// names of envs
	EnvENV                 = "CASSETTE_ENV"
	EnvNetworkInterface    = "CASSETTE_NETWORK_INTERFACE"
	EnvPort                = "CASSETTE_PORT"
	EnvAppURL              = "CASSETTE_APP_URL"
	EnvSecret              = "CASSETTE_SECRET"
	EnvMongoURI            = "CASSETTE_MONGODB_URI"
	EnvSpotifyClientID     = "CASSETTE_SPOTIFY_CLIENT_ID"
	EnvSpotifyClientSecret = "CASSETTE_SPOTIFY_CLIENT_KEY"
)
