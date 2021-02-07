package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/florianloch/cassette/internal/constants"
	"github.com/florianloch/cassette/internal/persistence"
	"github.com/florianloch/cassette/internal/spotify"
	"github.com/rs/zerolog/hlog"
	spotifyAPI "github.com/zmb3/spotify"
)

func ActiveDevicesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	spotifyClient := ctx.Value(constants.FieldKeySpotifyClient).(spotify.SpotClient)

	playerDevices, err := spotify.ActiveSpotifyDevices(spotifyClient)

	if err != nil {
		hlog.FromRequest(r).Debug().Err(err).Msg("Could not fetch list of active devices.")
		http.Error(w, "Could not fetch list of active devices from Spotify!", http.StatusInternalServerError)
	}

	json, err := json.Marshal(playerDevices)
	if err != nil {
		hlog.FromRequest(r).Debug().
			Err(err).Interface("playerDevices", playerDevices).
			Msg("Could not serialize player devices.")
		http.Error(w, "Could not fetch list of active devices from Spotify!", http.StatusInternalServerError)
	}

	if err != nil {
		hlog.FromRequest(r).Debug().Err(err).Msg("Could not fetch list of active devices.")
		http.Error(w, "Could not fetch list of active devices from Spotify!", http.StatusInternalServerError)
	}

	respondWithJSON(w, r, json)
}

func PlayerStatesPostHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := ctx.Value(constants.FieldKeyUser).(*spotifyAPI.PrivateUser)
	spotifyClient := ctx.Value(constants.FieldKeySpotifyClient).(spotify.SpotClient)
	dao := ctx.Value(constants.FieldKeyDao).(persistence.PlayerStatesPersistor)
	slot, ok := ctx.Value(constants.FieldKeySlot).(int)
	if !ok {
		slot = -1
	}

	currentState, err := spotify.CurrentPlayerState(spotifyClient)
	if err != nil {
		hlog.FromRequest(r).Error().Err(err).Msg("Failed to get current state of player.")
		http.Error(w, "Could not retrieve player state from Spotify. Please make sure your device is playing and online.", http.StatusInternalServerError)
		return
	}

	playerStates, err := dao.LoadPlayerStates(user.ID)
	if err != nil {
		hlog.FromRequest(r).Error().Err(err).Msg("Failed loading player states from DB.")
		http.Error(w, "Could not retrieve player states from DB.", http.StatusInternalServerError)
		return
	}

	// replace, if < 0 then append a new slot
	if slot >= 0 {
		if slot >= len(playerStates) {
			http.Error(w, "'slot' is not in the range of existing slots.", http.StatusBadRequest)
			hlog.FromRequest(r).Debug().Int("slot", slot).Msg("Slot is out of range.")
			return
		}

		playerStates[slot] = currentState
	} else {
		playerStates = append(playerStates, currentState)
	}

	err = dao.SavePlayerStates(user.ID, playerStates)
	if err != nil {
		hlog.FromRequest(r).Error().
			Err(err).
			Interface("playerStates", playerStates).
			Msg("Could not persist player states in DB.")
		http.Error(w, "Could not persist player states in DB.", http.StatusInternalServerError)
		return
	}

	err = spotify.PausePlayer(spotifyClient)
	if err != nil {
		// No serious error, we do not need to tell the client
		hlog.FromRequest(r).Debug().Err(err).Msg("Could not pause player.")
	}

	w.WriteHeader(http.StatusCreated)
}

func PlayerStatesGetHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := ctx.Value(constants.FieldKeyUser).(*spotifyAPI.PrivateUser)
	dao := ctx.Value(constants.FieldKeyDao).(persistence.PlayerStatesPersistor)

	playerStates, err := dao.LoadPlayerStates(user.ID)
	if err != nil {
		hlog.FromRequest(r).Error().Err(err).Msg("Failed loading player states from DB.")
		http.Error(w, "Could not retrieve player states from DB.", http.StatusInternalServerError)
		return
	}

	enriched := enrichPlayerStates(playerStates)
	json, err := json.Marshal(enriched)
	if err != nil {
		hlog.FromRequest(r).Error().
			Err(err).
			Interface("playerStates", playerStates).
			Msg("Could not serialize player states to JSON.")
		http.Error(w, "Failed to provide player states as JSON.", http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, r, json)
}

func PlayerStatesDeleteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := ctx.Value(constants.FieldKeyUser).(*spotifyAPI.PrivateUser)
	dao := ctx.Value(constants.FieldKeyDao).(persistence.PlayerStatesPersistor)
	slot := ctx.Value(constants.FieldKeySlot).(int)

	playerStates, err := dao.LoadPlayerStates(user.ID)
	if err != nil {
		hlog.FromRequest(r).Error().Err(err).Msg("Failed loading player states from DB.")
		http.Error(w, "Could not retrieve player states from DB.", http.StatusInternalServerError)
		return
	}

	if slot >= len(playerStates) {
		hlog.FromRequest(r).Debug().
			Int("slot", slot).
			Interface("playerStates", playerStates).
			Msg("Unable to delete player state - slot out of range.")
		http.Error(w, "'slot' is not in the range of existing slots.", http.StatusInternalServerError)
		return
	}

	playerStates = append(playerStates[:slot], playerStates[slot+1:]...)

	err = dao.SavePlayerStates(user.ID, playerStates)
	if err != nil {
		hlog.FromRequest(r).Error().
			Err(err).
			Interface("playerStates", playerStates).
			Msg("Could not persist player states in DB.")
		http.Error(w, "Could not persist player states in DB.", http.StatusInternalServerError)
	}
}

func PlayerStatesRestoreHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := ctx.Value(constants.FieldKeyUser).(*spotifyAPI.PrivateUser)
	spotifyClient := ctx.Value(constants.FieldKeySpotifyClient).(spotify.SpotClient)
	dao := ctx.Value(constants.FieldKeyDao).(persistence.PlayerStatesPersistor)
	slot := ctx.Value(constants.FieldKeySlot).(int)

	deviceID := r.URL.Query().Get("deviceID")
	playerStates, err := dao.LoadPlayerStates(user.ID)
	if err != nil {
		hlog.FromRequest(r).Error().Err(err).Msg("Failed loading player states from DB.")
		http.Error(w, "Could not retrieve player states from DB.", http.StatusInternalServerError)
		return
	}

	if slot >= len(playerStates) {
		hlog.FromRequest(r).Debug().
			Int("slot", slot).
			Interface("playerStates", playerStates).
			Msg("Unable to delete player state. Slot out of range.")
		http.Error(w, "'slot' is not in the range of existing slots.", http.StatusBadRequest)
		return
	}

	err = spotify.PausePlayer(spotifyClient)
	if err != nil {
		// No serious error, we do not need to tell the client, he might notice anyway
		hlog.FromRequest(r).Debug().Err(err).Msg("Could not pause player.")
	}

	stateToRestore := playerStates[slot]

	err = spotify.RestorePlayerState(spotifyClient, stateToRestore, deviceID)
	if err != nil {
		hlog.FromRequest(r).Debug().
			Err(err).
			Int("slot", slot).
			Str("deviceID", deviceID).
			Interface("stateToRestore", stateToRestore).
			Msg("Could not restore player state.")
		http.Error(w, "Could not restore player state. Please check that there is at least one active device.", http.StatusBadRequest)
	}
}

func UserExportHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := ctx.Value(constants.FieldKeyUser).(*spotifyAPI.PrivateUser)
	dao := ctx.Value(constants.FieldKeyDao).(persistence.PlayerStatesPersistor)

	json, err := dao.FetchJSONDump(user.ID)
	if err != nil {
		if errors.Is(err, persistence.ErrUserNotFound) {
			hlog.FromRequest(r).Debug().Msg("User requested to exports her/his data - but nothing found in DB.")
			http.Error(w, "No data stored in db for this user.", http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			hlog.FromRequest(r).Debug().Err(err).Msg("Failed exporting user data.")
		}

		return
	}

	respondWithJSON(w, r, json)
}

func UserDeleteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := ctx.Value(constants.FieldKeyUser).(*spotifyAPI.PrivateUser)
	dao := ctx.Value(constants.FieldKeyDao).(persistence.PlayerStatesPersistor)

	err := dao.DeleteUserRecord(user.ID)
	if err != nil {
		if errors.Is(err, persistence.ErrUserNotFound) {
			hlog.FromRequest(r).Debug().Msg("User requested to delete her/his data - but nothing found in DB.")
			http.Error(w, "No data stored in db for this user.", http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			hlog.FromRequest(r).Debug().Err(err).Msg("Failed deleting user data.")
		}
	}
}

func enrichPlayerStates(playerStates []*persistence.PlayerState) []*enrichedPlayerState {
	enrichedPlayerStates := make([]*enrichedPlayerState, len(playerStates))

	for i, playerState := range playerStates {
		enrichedPlayerStates[i] = &enrichedPlayerState{
			PlayerState:   playerState,
			LinkToContext: spotify.LinkToContext(playerState.PlaybackContextURI),
		}
	}

	return enrichedPlayerStates
}

type enrichedPlayerState struct {
	*persistence.PlayerState
	LinkToContext string `json:"linkToContext"` // link to open context in Spotify
}

func respondWithJSON(w http.ResponseWriter, r *http.Request, json []byte) {
	w.Header().Set("Content-Type", "application/json")
	bytesWritten, err := w.Write(json)
	if err != nil {
		hlog.FromRequest(r).Error().
			Err(err).
			Int("bytesWritten", bytesWritten).
			Str("json", string(json)).
			Msgf("Failed to write JSON response.")
	}
}
