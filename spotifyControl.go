package main

import (
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

func storeCurrentPlayerState(client *spotify.Client, userID *string) {
	var currentlyPlaying, err = client.PlayerCurrentlyPlaying()

	if err != nil {
		log.Fatal("Could not read the current player state!", err)
	}

	var playerStates = playerStatesDAO.LoadPlayerStates(*userID)
	playerStates.List[0] = playerStateFromCurrentlyPlaying(currentlyPlaying)
	playerStatesDAO.SavePlayerStates(playerStates)

	log.Println("Persisted current playing state:", currentlyPlaying)

	client.Pause()
}

func restorePlayerState(client *spotify.Client, userID *string) {
	var playerStates = playerStatesDAO.LoadPlayerStates(*userID)

	// if !ok {
	// 	log.Println("There is no last state for this user (" + *userID + ")!")
	// }

	var stateToLoad = playerStates.List[0]

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
}

func playerStateFromCurrentlyPlaying(currentlyPlaying *spotify.CurrentlyPlaying) *persistence.PlayerState {
	return &persistence.PlayerState{PlaybackContextURI: string(currentlyPlaying.PlaybackContext.URI), PlaybackItemURI: string(currentlyPlaying.Item.URI), Progress: currentlyPlaying.Progress}
}
