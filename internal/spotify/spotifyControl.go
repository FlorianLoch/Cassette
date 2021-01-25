package spotify

import (
	"errors"
	"log"

	constants "github.com/florianloch/cassette/internal"
	"github.com/florianloch/cassette/internal/persistence"

	"github.com/zmb3/spotify"
)

func isContextResumable(playbackContext spotify.PlaybackContext) bool {
	t := playbackContext.Type

	return t == "album" || t == "playlist"
}

func CurrentPlayerState(client *spotify.Client) (*persistence.PlayerState, error) {
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

func PausePlayer(client *spotify.Client) error {
	return client.Pause()
}

func RestorePlayerState(client *spotify.Client, stateToLoad *persistence.PlayerState, deviceID string) error {
	var err = client.Shuffle(stateToLoad.ShuffleActivated)
	if err != nil {
		return err
	}

	if stateToLoad.Progress >= constants.JumpBackNSeconds*1e3 {
		stateToLoad.Progress -= constants.JumpBackNSeconds * 1e3
	}

	var contextURI = spotify.URI(stateToLoad.PlaybackContextURI)
	var itemURI = spotify.URI(stateToLoad.PlaybackItemURI)
	var spotifyPlayOptions = &spotify.PlayOptions{
		PlaybackContext: &contextURI,
		PlaybackOffset:  &spotify.PlaybackOffset{URI: itemURI},
		PositionMs:      stateToLoad.Progress,
	}

	var id spotify.ID
	if deviceID == "" {
		var err error
		id, err = currentDeviceForPlayback(client)
		if err != nil {
			return err
		}
	} else {
		id = spotify.ID(deviceID)
	}

	spotifyPlayOptions.DeviceID = &id

	client.PlayOpt(spotifyPlayOptions)

	err = client.PlayOpt(spotifyPlayOptions)
	if err != nil {
		return err
	}

	return nil
}

func currentDeviceForPlayback(client *spotify.Client) (spotify.ID, error) {
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

func playerStateFromCurrentlyPlaying(currentlyPlaying *spotify.CurrentlyPlaying, shuffleActivated bool) *persistence.PlayerState {
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

func ActiveSpotifyDevices(client *spotify.Client) ([]CondensedPlayerDevice, error) {
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
