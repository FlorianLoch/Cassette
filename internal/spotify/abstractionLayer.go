package spotify

import (
	"net/http"

	spotifyAPI "github.com/zmb3/spotify"
	"golang.org/x/oauth2"
)

type SpotAuthenticator interface {
	AuthURL(state string) string
	NewClient(token *oauth2.Token) spotifyAPI.Client
	SetAuthInfo(clientID, secretKey string)
	Token(state string, r *http.Request) (*oauth2.Token, error)
}

type SpotClient interface {
	CurrentUser() (*spotifyAPI.PrivateUser, error)
	GetAlbumTracksOpt(id spotifyAPI.ID, opt *spotifyAPI.Options) (*spotifyAPI.SimpleTrackPage, error)
	GetPlaylistOpt(playlistID spotifyAPI.ID, fields string) (*spotifyAPI.FullPlaylist, error)
	GetPlaylistTracksOpt(playlistID spotifyAPI.ID, opt *spotifyAPI.Options, fields string) (*spotifyAPI.PlaylistTrackPage, error)
	Pause() error
	PlayerState() (*spotifyAPI.PlayerState, error)
	PlayerDevices() ([]spotifyAPI.PlayerDevice, error)
	PlayOpt(opt *spotifyAPI.PlayOptions) error
	Shuffle(shuffle bool) error
}
