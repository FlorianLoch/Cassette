package main

import (
	"log"

	"github.com/zmb3/spotify"
)

var (
	persistence = make(map[string]*spotify.CurrentlyPlaying)
)

func storeCurrentPlayerState(client *spotify.Client, userID *string) {
	var currentlyPlaying, err = client.PlayerCurrentlyPlaying()

	if err != nil {
		log.Fatal("Could not read the current player state!", err)
	}

	persistence[*userID] = currentlyPlaying

	log.Println("Persisted current playing state:", currentlyPlaying)
}

func restorePlayerState(client *spotify.Client, userID *string) {
	var lastState, ok = persistence[*userID]

	if !ok {
		log.Println("There is no last state for this user (" + *userID + ")!")
	}

	log.Println("Trying to restore the last state:", lastState)

	client.PlayOpt(&spotify.PlayOptions{
		PlaybackContext: &lastState.PlaybackContext.URI,
		PlaybackOffset:  &spotify.PlaybackOffset{URI: lastState.Item.URI},
	})

	client.Seek(lastState.Progress)
}
