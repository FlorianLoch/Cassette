package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/florianloch/cassette/internal/persistence"
	"github.com/florianloch/cassette/internal/spotify"
	"github.com/rs/zerolog/log"
	spotifyAPI "github.com/zmb3/spotify"
)

// TODO: Move to constants file
const (
	fieldDao           = "dao"
	fieldSlot          = "slot"
	fieldUser          = "user"
	fieldSpotifyClient = "spotifyClient"
)

func ActiveDevicesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	spotifyClient := ctx.Value(fieldSpotifyClient).(*spotifyAPI.Client)

	playerDevices, err := spotify.ActiveSpotifyDevices(spotifyClient)

	if err != nil {
		log.Debug().Err(err).Msg("Could not fetch list of active devices.")
		http.Error(w, "Could not fetch list of active devices from Spotify!", http.StatusInternalServerError)
	}

	json, err := json.Marshal(playerDevices)
	if err != nil {
		log.Debug().Err(err).Interface("playerDevices", playerDevices).Msg("Could not serialize player devices.")
		http.Error(w, "Could not fetch list of active devices from Spotify!", http.StatusInternalServerError)
	}

	if err != nil {
		log.Debug().Err(err).Msg("Could not fetch list of active devices.")
		http.Error(w, "Could not fetch list of active devices from Spotify!", http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
	w.Write(json)
}

func StorePostHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := ctx.Value(fieldUser).(*spotifyAPI.PrivateUser)
	spotifyClient := ctx.Value(fieldSpotifyClient).(*spotifyAPI.Client)
	dao := ctx.Value(fieldDao).(*persistence.PlayerStatesDAO)
	slot, ok := ctx.Value(fieldSlot).(int)
	if !ok {
		slot = -1
	}

	var currentState, err = spotify.CurrentPlayerState(spotifyClient)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get current state of player.")
		http.Error(w, "Could not retrieve player state from Spotify. Please make sure your device is playing and online.", http.StatusInternalServerError)
		return
	}

	playerStates, err := dao.LoadPlayerStates(user.ID)
	if err != nil {
		log.Error().Err(err).Msg("Failed loading player states from DB.")
		http.Error(w, "Could not retrieve player states from DB.", http.StatusInternalServerError)
		return
	}

	// replace, if < 0 then append a new slot
	if slot >= 0 {
		if slot >= len(playerStates) {
			http.Error(w, "'slot' is not in the range of existing slots.", http.StatusBadRequest)
			log.Debug().Int("slot", slot).Msg("Slot is out of range.")
			return
		}

		playerStates[slot] = currentState
	} else {
		playerStates = append(playerStates, currentState)
	}

	err = dao.SavePlayerStates(user.ID, playerStates)
	if err != nil {
		log.Error().Err(err).Interface("playerStates", playerStates).Msg("Could not persist player states in DB.")
		http.Error(w, "Could not persist player states in DB.", http.StatusInternalServerError)
		return
	}

	err = spotify.PausePlayer(spotifyClient)
	if err != nil {
		// No serious error, we do not need to tell the client
		log.Debug().Err(err).Msg("Could not pause player.")
	}

	w.WriteHeader(http.StatusCreated)
}

func StoreGetHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := ctx.Value(fieldUser).(*spotifyAPI.PrivateUser)
	dao := ctx.Value(fieldDao).(*persistence.PlayerStatesDAO)

	var playerStates, err = dao.LoadPlayerStates(user.ID)
	if err != nil {
		log.Error().Err(err).Msg("Failed loading player states from DB.")
		http.Error(w, "Could not retrieve player states from DB.", http.StatusInternalServerError)
		return
	}

	enriched := enrichPlayerStates(playerStates)
	json, err := json.Marshal(enriched)
	if err != nil {
		log.Error().Err(err).Interface("playerStates", playerStates).Msg("Could not serialize player states to JSON.")
		http.Error(w, "Failed to provide player states as JSON.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

func StoreDeleteHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Add a note that it is ensure that all values are set when these handlers are called?
	ctx := r.Context()
	user := ctx.Value(fieldUser).(*spotifyAPI.PrivateUser)
	dao := ctx.Value(fieldDao).(*persistence.PlayerStatesDAO)
	slot := ctx.Value(fieldSlot).(int)

	playerStates, err := dao.LoadPlayerStates(user.ID)
	if err != nil {
		log.Error().Err(err).Msg("Failed loading player states from DB.")
		http.Error(w, "Could not retrieve player states from DB.", http.StatusInternalServerError)
		return
	}

	if slot >= len(playerStates) {
		log.Debug().Int("slot", slot).Interface("playerStates", playerStates).Msg("Unable to delete player state - slot out of range.")
		http.Error(w, "'slot' is not in the range of existing slots.", http.StatusInternalServerError)
		return
	}

	playerStates = append(playerStates[:slot], playerStates[slot+1:]...)

	err = dao.SavePlayerStates(user.ID, playerStates)
	if err != nil {
		log.Error().Err(err).Interface("playerStates", playerStates).Msg("Could not persist player states in DB.")
		http.Error(w, "Could not persist player states in DB.", http.StatusInternalServerError)
	}
}

func RestoreHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := ctx.Value(fieldUser).(*spotifyAPI.PrivateUser)
	spotifyClient := ctx.Value(fieldSpotifyClient).(*spotifyAPI.Client)
	dao := ctx.Value(fieldDao).(*persistence.PlayerStatesDAO)
	slot := ctx.Value(fieldSlot).(int)

	var deviceID = r.URL.Query().Get("deviceID")
	playerStates, err := dao.LoadPlayerStates(user.ID)
	if err != nil {
		log.Error().Err(err).Msg("Failed loading player states from DB.")
		http.Error(w, "Could not retrieve player states from DB.", http.StatusInternalServerError)
		return
	}

	if slot >= len(playerStates) {
		log.Debug().Int("slot", slot).Interface("playerStates", playerStates).Msg("Unable to delete player state - slot out of range.")
		http.Error(w, "'slot' is not in the range of existing slots.", http.StatusInternalServerError)
		return
	}

	err = spotify.PausePlayer(spotifyClient)
	if err != nil {
		// No serious error, we do not need to tell the client
		log.Debug().Err(err).Msg("Could not pause player.")
	}

	var stateToRestore = playerStates[slot]

	err = spotify.RestorePlayerState(spotifyClient, stateToRestore, deviceID)
	if err != nil {
		log.Debug().Err(err).Int("slot", slot).Str("deviceID", deviceID).Interface("stateToRestore", stateToRestore).Msg("Could not restore player state.")
		http.Error(w, "Could not restore player state. Please check whether the given slot is valid and that there is at least one active device.", http.StatusBadRequest)
	}
}

func UserExportHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := ctx.Value(fieldUser).(*spotifyAPI.PrivateUser)
	dao := ctx.Value(fieldDao).(*persistence.PlayerStatesDAO)

	json, err := dao.FetchJSONDump(user.ID)
	if err != nil {
		if errors.Is(err, persistence.ErrUserNotFound) {
			log.Debug().Msg("User requested to exports her/his data - but nothing found in DB.")
			http.Error(w, "No data stored in db for this user.", http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Debug().Err(err).Msg("Failed exporting user data.")
		}

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

func UserDeleteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := ctx.Value(fieldUser).(*spotifyAPI.PrivateUser)
	dao := ctx.Value(fieldDao).(*persistence.PlayerStatesDAO)

	err := dao.DeleteUserRecord(user.ID)
	if err != nil {
		if errors.Is(err, persistence.ErrUserNotFound) {
			log.Debug().Msg("User requested to delete her/his data - but nothing found in DB.")
			http.Error(w, "No data stored in db for this user.", http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Debug().Err(err).Msg("Failed deleting user data.")
		}
	}
}

func enrichPlayerStates(playerStates []*persistence.PlayerState) []*enrichedPlayerState {
	enrichedPlayerStates := make([]*enrichedPlayerState, len(playerStates))

	for i, playerState := range playerStates {
		enrichedPlayerStates[i] = &enrichedPlayerState{
			PlayerState:   playerState,
			LinkToContext: linkToContext(playerState.PlaybackContextURI),
		}
	}

	return enrichedPlayerStates
}

func linkToContext(playbackContextURI string) string {
	splits := strings.Split(playbackContextURI, ":")

	if len(splits) != 3 {
		log.Error().Str("playbackContextURI", playbackContextURI).Interface("splits", splits).Msg("Splitting context URI did not result in 3 parts.")

		return ""
	}

	return fmt.Sprintf("https://open.spotify.com/%s/%s", splits[1], splits[2])
}

type enrichedPlayerState struct {
	*persistence.PlayerState
	LinkToContext string
}
