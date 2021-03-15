package spotify

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	spotifyAPI "github.com/zmb3/spotify"

	"github.com/florianloch/cassette/internal/constants"
	"github.com/florianloch/cassette/internal/persistence"
)

const (
	pagingLimit = 50
)

var (
	ErrTrackNotFoundInContext    = errors.New("could not find track in context")
	ErrNoActiveDeviceForPlayback = errors.New("no (active) device available for playback")
	ErrContextNotSuspendable     = errors.New("the current context cannot be restored! It is only possible to store playing positions in albums and playlists")
)

func isContextSuspendable(playbackContext spotifyAPI.PlaybackContext) bool {
	t := playbackContext.Type

	return t == "album" || t == "playlist"
}

func CurrentPlayerState(client SpotClient) (*persistence.PlayerState, error) {
	playerState, err := client.PlayerState()
	var shuffleActivated bool
	if err != nil {
		return nil, fmt.Errorf("could not read whats currently playing: %w", err)
	}
	shuffleActivated = playerState.ShuffleState

	currentlyPlaying := &playerState.CurrentlyPlaying

	//Check whether this position could possibly restored afterwards
	if !isContextSuspendable(currentlyPlaying.PlaybackContext) {
		return nil, ErrContextNotSuspendable
	}

	item := currentlyPlaying.Item
	joinedArtists := ""
	for idx, artist := range item.Artists {
		joinedArtists += artist.Name
		if idx < len(item.Artists)-1 {
			joinedArtists += ", "
		}
	}

	// Ensure there are two URLs in the 'Images' slice
	images := item.Album.Images
	if len(images) == 0 {
		// Kind of an assert, should not happen. In case it does it's not too important though
		log.Error().Interface("item", item).Msg("No image URL provided for currently playing item.")

		item.Album.Images = append(images, spotifyAPI.Image{
			URL: "",
		})
	}
	if len(images) == 1 {
		log.Error().Interface("item", item).Msg("Just one URL provided for currently playing item.")

		item.Album.Images = append(images, images[0])
	}

	trackIndex, totalTracks, err := indexOfCurrentTrack(currentlyPlaying, client)
	if err != nil {
		// No need to stop processing this request because of this error...
		log.Error().Err(err).Interface("item", item).Msg("Could not get index of track in context.")
	}

	linkToContext, ok := currentlyPlaying.PlaybackContext.ExternalURLs["spotify"]
	if !ok {
		// No need to stop processing this request because of this error...
		log.Error().
			Err(err).
			Interface("playbackContext", currentlyPlaying.PlaybackContext).
			Msg("Could not get link to context from response.")
	}

	playlistName := ""
	if currentlyPlaying.PlaybackContext.Type == "playlist" {
		playlistID := idOfContext(currentlyPlaying)
		playlist, err := client.GetPlaylistOpt(playlistID, "name")
		if err != nil {
			// No need to stop processing this request because of this error...
			log.Error().Err(err).Str("playlistID", string(playlistID)).Msg("Could not get name of playlist.")
		} else {
			playlistName = playlist.Name
		}
	}

	return &persistence.PlayerState{
		PlaybackContextURI: string(currentlyPlaying.PlaybackContext.URI),
		PlaybackItemURI:    string(item.URI),
		LinkToContext:      linkToContext,
		ContextType:        currentlyPlaying.PlaybackContext.Type,
		PlaylistName:       playlistName,
		AlbumArtLargeURL:   item.Album.Images[0].URL,
		AlbumArtMediumURL:  item.Album.Images[1].URL,
		TrackName:          item.Name,
		AlbumName:          item.Album.Name,
		ArtistName:         joinedArtists,
		TrackIndex:         trackIndex,
		TotalTracks:        totalTracks,
		Progress:           currentlyPlaying.Progress,
		Duration:           item.Duration,
		ShuffleActivated:   shuffleActivated,
		SuspendedAtTs:      time.Now().Unix(),
	}, nil
}

func RestorePlayerState(client SpotClient, stateToLoad *persistence.PlayerState, deviceID string) error {
	err := client.Shuffle(stateToLoad.ShuffleActivated)
	if err != nil {
		return err
	}

	stateToLoad.Progress -= min(stateToLoad.Progress, constants.JumpBackNSeconds*1e3)

	contextURI := spotifyAPI.URI(stateToLoad.PlaybackContextURI)
	itemURI := spotifyAPI.URI(stateToLoad.PlaybackItemURI)
	spotifyPlayOptions := &spotifyAPI.PlayOptions{
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
		return "", ErrNoActiveDeviceForPlayback
	}

	for _, device := range devices {
		if device.Active {
			return device.ID, nil
		}
	}

	return devices[0].ID, nil
}

func indexOfCurrentTrack(currentlyPlaying *spotifyAPI.CurrentlyPlaying, client SpotClient) (int, int, error) {
	typ := currentlyPlaying.PlaybackContext.Type

	// Has to be "album" or "playlist" - this should be ensured upstream.
	// So this check is basically an assert
	isAlbum := typ == "album"
	isPlaylist := typ == "playlist"
	if !isAlbum && !isPlaylist {
		log.Panic().Str("type", typ).Msg("called with context neither being 'album' nor 'playlist'")
	}

	trackID := currentlyPlaying.Item.ID
	contextID := idOfContext(currentlyPlaying)

	offset := 0
	limit := pagingLimit
	options := spotifyAPI.Options{
		Limit:  &limit,
		Offset: &offset,
	}
	var index int
	var total int

	for {
		if isAlbum {
			page, err := client.GetAlbumTracksOpt(contextID, &options)
			if err != nil {
				return -1, -1, err
			}

			index = findTrackInSimpleTrackPages(trackID, page)
			total = page.Total
		} else {
			page, err := client.GetPlaylistTracksOpt(contextID, &options, "total,limit,items(track(id))")
			if err != nil {
				return -1, -1, err
			}

			index = findTrackInPlaylistTrackPage(trackID, page)
			total = page.Total
		}

		if index >= 0 {
			index += offset + 1 // because the user probably does not expect zero-based counting
			return index, total, nil
		}

		offset += limit
		if offset >= total {
			return -1, -1, ErrTrackNotFoundInContext
		}
	}
}

func idOfContext(currentlyPlaying *spotifyAPI.CurrentlyPlaying) spotifyAPI.ID {
	uri := currentlyPlaying.PlaybackContext.URI
	splits := strings.Split(string(uri), ":")
	return spotifyAPI.ID(splits[len(splits)-1])
}

func findTrackInSimpleTrackPages(trackID spotifyAPI.ID, page *spotifyAPI.SimpleTrackPage) int {
	for i, track := range page.Tracks {
		if track.ID == trackID {
			return i
		}
	}

	return -1
}

func findTrackInPlaylistTrackPage(trackID spotifyAPI.ID, page *spotifyAPI.PlaylistTrackPage) int {
	for i, track := range page.Tracks {
		if track.Track.ID == trackID {
			return i
		}
	}

	return -1
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
