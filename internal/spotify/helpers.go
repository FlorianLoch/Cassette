package spotify

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
)

func LinkToContext(playbackContextURI string) string {
	splits := strings.Split(playbackContextURI, ":")

	if len(splits) != 3 {
		log.Error().Str("playbackContextURI", playbackContextURI).Interface("splits", splits).Msg("Splitting context URI did not result in 3 parts.")

		return ""
	}

	return fmt.Sprintf("https://open.spotify.com/%s/%s", splits[1], splits[2])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
