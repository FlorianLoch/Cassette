package spotify

// DO NOT EDIT!
// This code is generated with http://github.com/hexdigest/gowrap tool
// using resilienceWrapper.tmpl template

//go:generate gowrap gen -p github.com/florianloch/cassette/internal/spotify -i SpotClient -t resilienceWrapper.tmpl -o resilientSpotClient.go

// To be used with https://github.com/hexdigest/gowrap
// Based on https://github.com/hexdigest/gowrap/blob/a00b5e810bdf0db43652c86216d4dfd2fc8c9afc/templates/retry
import (
	"time"

	"github.com/rs/zerolog/log"
	spotifyAPI "github.com/zmb3/spotify"
)

// SpotClientWithRetry implements SpotClient interface instrumented with retries
type SpotClientWithRetry struct {
	SpotClient
	_retryCount int
	_waitFor    time.Duration
}

// NewSpotClientWithRetry returns SpotClientWithRetry
func NewSpotClientWithRetry(base SpotClient, retryCount int, waitFor time.Duration) SpotClientWithRetry {
	return SpotClientWithRetry{
		SpotClient:  base,
		_retryCount: retryCount,
		_waitFor:    waitFor,
	}
}

// CurrentUser implements SpotClient
func (_d SpotClientWithRetry) CurrentUser() (pp1 *spotifyAPI.PrivateUser, err error) {
	pp1, err = _d.SpotClient.CurrentUser()
	if err == nil || _d._retryCount < 1 {
		return
	}
	_ticker := time.NewTicker(_d._waitFor)
	defer _ticker.Stop()
	for _i := 0; _i < _d._retryCount && err != nil; _i++ {
		<-_ticker.C
		pp1, err = _d.SpotClient.CurrentUser()
		if err != nil {
			log.Warn().Msgf("Call to 'CurrentUser' only succeeded due to retrying %d time(s).", _i+1)
		}
	}
	return
}

// GetAlbumTracksOpt implements SpotClient
func (_d SpotClientWithRetry) GetAlbumTracksOpt(id spotifyAPI.ID, opt *spotifyAPI.Options) (sp1 *spotifyAPI.SimpleTrackPage, err error) {
	sp1, err = _d.SpotClient.GetAlbumTracksOpt(id, opt)
	if err == nil || _d._retryCount < 1 {
		return
	}
	_ticker := time.NewTicker(_d._waitFor)
	defer _ticker.Stop()
	for _i := 0; _i < _d._retryCount && err != nil; _i++ {
		<-_ticker.C
		sp1, err = _d.SpotClient.GetAlbumTracksOpt(id, opt)
		if err != nil {
			log.Warn().Msgf("Call to 'GetAlbumTracksOpt' only succeeded due to retrying %d time(s).", _i+1)
		}
	}
	return
}

// GetPlaylistOpt implements SpotClient
func (_d SpotClientWithRetry) GetPlaylistOpt(playlistID spotifyAPI.ID, fields string) (fp1 *spotifyAPI.FullPlaylist, err error) {
	fp1, err = _d.SpotClient.GetPlaylistOpt(playlistID, fields)
	if err == nil || _d._retryCount < 1 {
		return
	}
	_ticker := time.NewTicker(_d._waitFor)
	defer _ticker.Stop()
	for _i := 0; _i < _d._retryCount && err != nil; _i++ {
		<-_ticker.C
		fp1, err = _d.SpotClient.GetPlaylistOpt(playlistID, fields)
		if err != nil {
			log.Warn().Msgf("Call to 'GetPlaylistOpt' only succeeded due to retrying %d time(s).", _i+1)
		}
	}
	return
}

// GetPlaylistTracksOpt implements SpotClient
func (_d SpotClientWithRetry) GetPlaylistTracksOpt(playlistID spotifyAPI.ID, opt *spotifyAPI.Options, fields string) (pp1 *spotifyAPI.PlaylistTrackPage, err error) {
	pp1, err = _d.SpotClient.GetPlaylistTracksOpt(playlistID, opt, fields)
	if err == nil || _d._retryCount < 1 {
		return
	}
	_ticker := time.NewTicker(_d._waitFor)
	defer _ticker.Stop()
	for _i := 0; _i < _d._retryCount && err != nil; _i++ {
		<-_ticker.C
		pp1, err = _d.SpotClient.GetPlaylistTracksOpt(playlistID, opt, fields)
		if err != nil {
			log.Warn().Msgf("Call to 'GetPlaylistTracksOpt' only succeeded due to retrying %d time(s).", _i+1)
		}
	}
	return
}

// Pause implements SpotClient
func (_d SpotClientWithRetry) Pause() (err error) {
	err = _d.SpotClient.Pause()
	if err == nil || _d._retryCount < 1 {
		return
	}
	_ticker := time.NewTicker(_d._waitFor)
	defer _ticker.Stop()
	for _i := 0; _i < _d._retryCount && err != nil; _i++ {
		<-_ticker.C
		err = _d.SpotClient.Pause()
		if err != nil {
			log.Warn().Msgf("Call to 'Pause' only succeeded due to retrying %d time(s).", _i+1)
		}
	}
	return
}

// PlayOpt implements SpotClient
func (_d SpotClientWithRetry) PlayOpt(opt *spotifyAPI.PlayOptions) (err error) {
	err = _d.SpotClient.PlayOpt(opt)
	if err == nil || _d._retryCount < 1 {
		return
	}
	_ticker := time.NewTicker(_d._waitFor)
	defer _ticker.Stop()
	for _i := 0; _i < _d._retryCount && err != nil; _i++ {
		<-_ticker.C
		err = _d.SpotClient.PlayOpt(opt)
		if err != nil {
			log.Warn().Msgf("Call to 'PlayOpt' only succeeded due to retrying %d time(s).", _i+1)
		}
	}
	return
}

// PlayerDevices implements SpotClient
func (_d SpotClientWithRetry) PlayerDevices() (pa1 []spotifyAPI.PlayerDevice, err error) {
	pa1, err = _d.SpotClient.PlayerDevices()
	if err == nil || _d._retryCount < 1 {
		return
	}
	_ticker := time.NewTicker(_d._waitFor)
	defer _ticker.Stop()
	for _i := 0; _i < _d._retryCount && err != nil; _i++ {
		<-_ticker.C
		pa1, err = _d.SpotClient.PlayerDevices()
		if err != nil {
			log.Warn().Msgf("Call to 'PlayerDevices' only succeeded due to retrying %d time(s).", _i+1)
		}
	}
	return
}

// PlayerState implements SpotClient
func (_d SpotClientWithRetry) PlayerState() (pp1 *spotifyAPI.PlayerState, err error) {
	pp1, err = _d.SpotClient.PlayerState()
	if err == nil || _d._retryCount < 1 {
		return
	}
	_ticker := time.NewTicker(_d._waitFor)
	defer _ticker.Stop()
	for _i := 0; _i < _d._retryCount && err != nil; _i++ {
		<-_ticker.C
		pp1, err = _d.SpotClient.PlayerState()
		if err != nil {
			log.Warn().Msgf("Call to 'PlayerState' only succeeded due to retrying %d time(s).", _i+1)
		}
	}
	return
}

// Shuffle implements SpotClient
func (_d SpotClientWithRetry) Shuffle(shuffle bool) (err error) {
	err = _d.SpotClient.Shuffle(shuffle)
	if err == nil || _d._retryCount < 1 {
		return
	}
	_ticker := time.NewTicker(_d._waitFor)
	defer _ticker.Stop()
	for _i := 0; _i < _d._retryCount && err != nil; _i++ {
		<-_ticker.C
		err = _d.SpotClient.Shuffle(shuffle)
		if err != nil {
			log.Warn().Msgf("Call to 'Shuffle' only succeeded due to retrying %d time(s).", _i+1)
		}
	}
	return
}
