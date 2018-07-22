package main

import (
	"errors"
	"log"
	"os"
	"strings"

	"github.com/florianloch/audioBookHelperForSpotify/persistence"
	"github.com/zmb3/spotify"
)

var (
	playerStatesDAO = persistence.NewPlayerStatesDAO(mongoURI)
	mongoURI        = strings.TrimSpace(os.Getenv("mongo_db_uri"))
)

func storeCurrentPlayerState(client *spotify.Client, userID *string, slot int) error {
	var currentlyPlaying, err = client.PlayerCurrentlyPlaying()

	if err != nil {
		log.Println("Could not read the current player state!", err)
		return errors.New("could not read the current player state")
	}

	var playerStates = playerStatesDAO.LoadPlayerStates(*userID)
	var currentState = playerStateFromCurrentlyPlaying(currentlyPlaying)

	// replace, if < 0 then append a new slot
	if slot > 0 {
		if slot >= len(playerStates.List) {
			return errors.New("'slot' is not in the range of exisiting slots")
		}

		playerStates.List[slot] = currentState
	} else {
		playerStates.List = append(playerStates.List, currentState)
	}

	playerStatesDAO.SavePlayerStates(playerStates)

	log.Println("Persisted current playing state:", currentlyPlaying)

	client.Pause()

	return nil
}

func restorePlayerState(client *spotify.Client, userID *string, slot int) error {
	var playerStates = playerStatesDAO.LoadPlayerStates(*userID)

	// if !ok {
	// 	log.Println("There is no last state for this user (" + *userID + ")!")
	// }

	if slot >= len(playerStates.List) || slot < 0 {
		return errors.New("'slot' is not in the range of exisiting slots")
	}

	var stateToLoad = playerStates.List[slot]

	log.Println("Trying to restore the last state:", stateToLoad)

	var contextURI = spotify.URI(stateToLoad.PlaybackContextURI)
	var itemURI = spotify.URI(stateToLoad.PlaybackItemURI)
	var spotifyPlayOptions = spotify.PlayOptions{
		PlaybackContext: &contextURI,
		PlaybackOffset:  &spotify.PlaybackOffset{URI: itemURI},
	}
	client.PlayOpt(&spotifyPlayOptions)

	log.Println(string(itemURI))

	client.Seek(stateToLoad.Progress)

	client.Play()

	return nil
}

func playerStateFromCurrentlyPlaying(currentlyPlaying *spotify.CurrentlyPlaying) *persistence.PlayerState {
	return &persistence.PlayerState{PlaybackContextURI: string(currentlyPlaying.PlaybackContext.URI), PlaybackItemURI: string(currentlyPlaying.Item.URI), Progress: currentlyPlaying.Progress}
}
