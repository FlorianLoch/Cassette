package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	constants "github.com/florianloch/cassette/internal"
	"github.com/florianloch/cassette/internal/persistence"
	"github.com/florianloch/cassette/internal/spotify"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/rs/zerolog/log"
	spotifyAPI "github.com/zmb3/spotify"
	"golang.org/x/oauth2"
)

func ActiveDevicesHandler(w http.ResponseWriter, r *http.Request, store *sessions.CookieStore, auth *spotifyAPI.Authenticator) {
	playerDevices, err := spotify.ActiveSpotifyDevices(spotifyClientFromRequest(r, store, auth))

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

func StorePostHandler(w http.ResponseWriter, r *http.Request, store *sessions.CookieStore, auth *spotifyAPI.Authenticator, dao *persistence.PlayerStatesDAO, slot int) {
	var userID = currentUser(r, store).ID
	var spotifyClient = spotifyClientFromRequest(r, store, auth)
	var currentState, err = spotify.CurrentPlayerState(spotifyClient)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get current state of player.")
		http.Error(w, "Could not retrieve player state from Spotify. Please make sure your device is playing and online.", http.StatusInternalServerError)
		return
	}

	playerStates, err := dao.LoadPlayerStates(userID)
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

	err = dao.SavePlayerStates(userID, playerStates)
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

func StoreGetHandler(w http.ResponseWriter, r *http.Request, store *sessions.CookieStore, dao *persistence.PlayerStatesDAO) {
	var playerStates, err = dao.LoadPlayerStates(currentUser(r, store).ID)
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

func StoreDeleteHandler(w http.ResponseWriter, r *http.Request, store *sessions.CookieStore, dao *persistence.PlayerStatesDAO) {
	var userID = currentUser(r, store).ID

	var slot, err = CheckSlotParameter(r)
	if err != nil {
		log.Debug().Err(err).Msg("Could not process request.")
		http.Error(w, "Could not process request. Please check whether the given slot is valid.", http.StatusBadRequest)
		return
	}

	playerStates, err := dao.LoadPlayerStates(userID)
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

	err = dao.SavePlayerStates(userID, playerStates)
	if err != nil {
		log.Error().Err(err).Interface("playerStates", playerStates).Msg("Could not persist player states in DB.")
		http.Error(w, "Could not persist player states in DB.", http.StatusInternalServerError)
	}
}

func RestoreHandler(w http.ResponseWriter, r *http.Request, store *sessions.CookieStore, auth *spotifyAPI.Authenticator, dao *persistence.PlayerStatesDAO) {
	var spotifyClient = spotifyClientFromRequest(r, store, auth)

	var slot, err = CheckSlotParameter(r)
	if err != nil {
		log.Debug().Err(err).Msg("Could not process request.")
		http.Error(w, "Could not process request. Please check whether the given slot is valid.", http.StatusBadRequest)
		return
	}

	var deviceID = r.URL.Query().Get("deviceID")
	var userID = currentUser(r, store).ID
	playerStates, err := dao.LoadPlayerStates(userID)
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

func UserExportHandler(w http.ResponseWriter, r *http.Request, store *sessions.CookieStore, dao *persistence.PlayerStatesDAO) {
	json, err := dao.FetchJSONDump(currentUser(r, store).ID)
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

func UserDeleteHandler(w http.ResponseWriter, r *http.Request, store *sessions.CookieStore, dao *persistence.PlayerStatesDAO) {
	err := dao.DeleteUserRecord(currentUser(r, store).ID)
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

func CheckSlotParameter(r *http.Request) (int, error) {
	var rawSlot, ok = mux.Vars(r)["slot"]

	if !ok {
		return -1, errors.New("query parameter 'slot' not found or more than one provided")
	}

	var slot, err = strconv.Atoi(rawSlot)
	if err != nil {
		return -1, errors.New("query parameter 'slot' is not a valid integer")
	}
	if slot < 0 {
		return -1, errors.New("query parameter 'slot' has to be >= 0")
	}

	return slot, nil
}

func spotifyClientFromRequest(r *http.Request, store *sessions.CookieStore, auth *spotifyAPI.Authenticator) *spotifyAPI.Client {
	session, _ := store.Get(r, constants.SessionCookieName)

	rawToken := session.Values["spotify-oauth-token"]

	return SpotifyClientFromToken(rawToken, auth)
}

func currentUser(r *http.Request, store *sessions.CookieStore) *spotifyAPI.PrivateUser {
	session, _ := store.Get(r, constants.SessionCookieName)

	rawUser := session.Values["user"]

	var user = &spotifyAPI.PrivateUser{}
	var ok = true
	if user, ok = rawUser.(*spotifyAPI.PrivateUser); !ok {
		// Fatal should be fine in this case as this is not an error that should ever occur.
		log.Fatal().Msg("Could not type-assert the stored user!")
	}

	return user
}

func SpotifyClientFromToken(rawToken interface{}, auth *spotifyAPI.Authenticator) *spotifyAPI.Client {
	var tok = &oauth2.Token{}
	var ok = true
	if tok, ok = rawToken.(*oauth2.Token); !ok {
		// Fatal should be fine in this case as this is not an error that should ever occur.
		log.Fatal().Interface("rawToken", rawToken).Msg("Could not type-assert the stored token!")
	}

	client := auth.NewClient(tok)

	return &client
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
