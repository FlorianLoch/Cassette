package main

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/florianloch/cassette/internal"
)

var (
	// Build flags set by Makefile
	gitVersion    string
	gitAuthorDate string
	buildDate     string
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	log.Info().Str("gitCommit", gitVersion).Str("gitDate", gitAuthorDate).Str("builtAt", buildDate).Msg("")

	internal.RunInProduction()
}
