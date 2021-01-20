package main

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/florianloch/spotistate/persistence"

	"github.com/zmb3/spotify"
)

func isContextResumable(playbackContext spotify.PlaybackContext) bool {
	t := playbackContext.Type

	return t == "album" || t == "playlist"
}

func storeCurrentPlayerState(client *spotify.Client, userID string, slot int) error {
	var currentlyPlaying, err = client.PlayerCurrentlyPlaying()
	if err != nil {
		log.Println("Could not read whats currently playing!", err)
		return errors.New("could not read whats currently playing")
	}

	//Check whether this position could possibly restored afterwards
	if !isContextResumable(currentlyPlaying.PlaybackContext) {
		return errors.New("the current context cannot be restored! It is only possible to store playing positions in albums and playlists")
	}

	playerState, err := client.PlayerState()
	var shuffleActivated = false
	if err == nil {
		shuffleActivated = playerState.ShuffleState
	} else {
		log.Println("Could not read the current player state, we assume shuffle is deactivated therefore.")
	}

	var playerStates = playerStatesDAO.LoadPlayerStates(userID)
	var currentState = playerStateFromCurrentlyPlaying(currentlyPlaying, shuffleActivated)

	// replace, if < 0 then append a new slot
	if slot >= 0 {
		if slot >= len(playerStates.States) {
			return errors.New("'slot' is not in the range of exisiting slots")
		}

		playerStates.States[slot] = currentState
	} else {
		playerStates.States = append(playerStates.States, currentState)
	}

	playerStatesDAO.SavePlayerStates(playerStates)

	log.Println("Persisted current playing state:", currentlyPlaying)

	client.Pause()

	return nil
}

func restorePlayerState(client *spotify.Client, userID string, slot int, deviceID string) error {
	var playerStates = playerStatesDAO.LoadPlayerStates(userID)

	if slot >= len(playerStates.States) || slot < 0 {
		return errors.New("'slot' is not in the range of exisiting slots")
	}

	var stateToLoad = playerStates.States[slot]

	log.Println("Trying to restore the last state on device '", deviceID, "': ", stateToLoad)

	client.Pause()

	var err = client.Shuffle(stateToLoad.ShuffleActivated)
	if err != nil {
		log.Println("Could not restore shuffle state: ", err)
	}

	if stateToLoad.Progress >= jumpBackNSeconds*1e3 {
		stateToLoad.Progress -= jumpBackNSeconds * 1e3
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
		id, err = getDeviceForPlayback(client)

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
		log.Println(err)
	}

	return nil
}

func getDeviceForPlayback(client *spotify.Client) (spotify.ID, error) {
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

func getActiveSpotifyDevices(client *spotify.Client) ([]byte, error) {
	devices, err := client.PlayerDevices()

	if err != nil {
		return nil, err
	}

	condensedDevices := make([]condensedPlayerDevice, len(devices))

	for i, device := range devices {
		condensedDevices[i] = condensedPlayerDevice{
			ID:     string(device.ID),
			Name:   device.Name,
			Active: device.Active,
		}
	}

	json, err := json.Marshal(condensedDevices)

	return json, err
}

type condensedPlayerDevice struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}
