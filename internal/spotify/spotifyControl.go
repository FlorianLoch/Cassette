package spotify

import (
	"errors"
	"log"

	"github.com/florianloch/cassette/internal/constants"
	"github.com/florianloch/cassette/internal/persistence"

	spotifyAPI "github.com/zmb3/spotify"
)

// TODO: Replace verbose var declaration with shorthand version for consistency

func isContextResumable(playbackContext spotifyAPI.PlaybackContext) bool {
	t := playbackContext.Type

	return t == "album" || t == "playlist"
}

func CurrentPlayerState(client SpotClient) (*persistence.PlayerState, error) {
	var currentlyPlaying, err = client.PlayerCurrentlyPlaying()
	if err != nil {
		log.Println("Could not read whats currently playing!", err)
		return nil, errors.New("could not read whats currently playing")
	}

	//Check whether this position could possibly restored afterwards
	if !isContextResumable(currentlyPlaying.PlaybackContext) {
		return nil, errors.New("the current context cannot be restored! It is only possible to store playing positions in albums and playlists")
	}

	playerState, err := client.PlayerState()
	var shuffleActivated = false
	if err == nil {
		shuffleActivated = playerState.ShuffleState
	} else {
		log.Println("Could not read the current player state, we assume shuffle is deactivated therefore.")
	}

	return playerStateFromCurrentlyPlaying(currentlyPlaying, shuffleActivated), nil
}

func PausePlayer(client SpotClient) error {
	return client.Pause()
}

func RestorePlayerState(client SpotClient, stateToLoad *persistence.PlayerState, deviceID string) error {
	var err = client.Shuffle(stateToLoad.ShuffleActivated)
	if err != nil {
		return err
	}

	if stateToLoad.Progress >= constants.JumpBackNSeconds*1e3 {
		stateToLoad.Progress -= constants.JumpBackNSeconds * 1e3
	}

	var contextURI = spotifyAPI.URI(stateToLoad.PlaybackContextURI)
	var itemURI = spotifyAPI.URI(stateToLoad.PlaybackItemURI)
	var spotifyPlayOptions = &spotifyAPI.PlayOptions{
		PlaybackContext: &contextURI,
		PlaybackOffset:  &spotifyAPI.PlaybackOffset{URI: itemURI},
		PositionMs:      stateToLoad.Progress,
	}

	var id spotifyAPI.ID
	if deviceID == "" {
		var err error
		id, err = currentDeviceForPlayback(client)
		if err != nil {
			return err
		}
	} else {
		id = spotifyAPI.ID(deviceID)
	}

	spotifyPlayOptions.DeviceID = &id

	client.PlayOpt(spotifyPlayOptions)

	err = client.PlayOpt(spotifyPlayOptions)
	if err != nil {
		return err
	}

	return nil
}

func currentDeviceForPlayback(client SpotClient) (spotifyAPI.ID, error) {
	devices, err := client.PlayerDevices()

	if err != nil {
		return "", err
	}

	if len(devices) == 0 {
		return "", errors.New("No (active) device available for playback!")
	}

	for _, device := range devices {
		if device.Active {
			return device.ID, nil
		}
	}

	return devices[0].ID, nil
}

func playerStateFromCurrentlyPlaying(currentlyPlaying *spotifyAPI.CurrentlyPlaying, shuffleActivated bool) *persistence.PlayerState {
	var item = currentlyPlaying.Item
	var joinedArtists = ""
	for idx, artist := range item.Artists {
		joinedArtists += artist.Name
		if idx < len(item.Artists)-1 {
			joinedArtists += ", "
		}
	}

	return &persistence.PlayerState{string(currentlyPlaying.PlaybackContext.URI), string(item.URI), item.Name, item.Album.Name, item.Album.Images[0].URL, joinedArtists, currentlyPlaying.Progress, item.Duration, shuffleActivated}
}

type CondensedPlayerDevice struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

func ActiveSpotifyDevices(client SpotClient) ([]CondensedPlayerDevice, error) {
	devices, err := client.PlayerDevices()

	if err != nil {
		return nil, err
	}

	condensedDevices := make([]CondensedPlayerDevice, len(devices))

	for i, device := range devices {
		condensedDevices[i] = CondensedPlayerDevice{
			ID:     string(device.ID),
			Name:   device.Name,
			Active: device.Active,
		}
	}

	return condensedDevices, nil
}
