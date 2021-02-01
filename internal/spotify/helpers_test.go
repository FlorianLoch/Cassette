package spotify

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLinkToContext(t *testing.T) {
	assert := assert.New(t)

	res := LinkToContext("spotify:playlist:37i9dQZF1DXa2SPUyWl8Y5")
	assert.Equal("https://open.spotify.com/playlist/37i9dQZF1DXa2SPUyWl8Y5", res)

	res = LinkToContext("spotify:album:08tZq3FDsspdU6ycn8Jl2o")
	assert.Equal("https://open.spotify.com/album/08tZq3FDsspdU6ycn8Jl2o", res)

	// in case of an unexpected format empty string should be returned
	res = LinkToContext("spotify:al:bum:08tZq3FDsspdU6ycn8Jl2o")
	assert.Equal("", res)
}
