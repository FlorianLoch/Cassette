package e2e_test

import (
	"net/http"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/golang/mock/gomock"
	"golang.org/x/oauth2"

	main "github.com/florianloch/cassette/internal"
	"github.com/florianloch/cassette/internal/constants"
	"github.com/florianloch/cassette/internal/e2e_test/mocks"
	"github.com/florianloch/cassette/internal/spotify"
)

const (
	snippedFromIndexPage = "We're sorry but Cassette doesn't work properly without JavaScript enabled."
)

func TestConsentCheck(t *testing.T) {
	e, ctrl, _, authMock, _ := beforeEach(t)
	defer ctrl.Finish()

	authMock.EXPECT().AuthURL(gomock.Any()).Times(1).Return("https://auth.spotify.com/some-redirect-url")

	// Requesting without consent token API routes should be reachable
	r := e.HEAD("/api/csrfToken").Expect()
	r.Header(constants.CSRFHeaderName).NotEmpty()
	r.Cookie(constants.CSRFCookieName).Value().NotEmpty()

	// Requesting index we will always be served the web app until we provide a valid consent cookie
	e.GET("/").Expect().Body().Contains(snippedFromIndexPage)

	// Check whether an invalid consent cookie gets handled well
	r = e.GET("/").WithCookie(constants.ConsentCookieName, "ancient cookie, tastes really bad").Expect()
	r.Header(constants.ConsentNoticeHeaderName).Equal("ATTENTION: consent not given yet.")
	r.Body().Contains(snippedFromIndexPage)

	// No try again with a valid cookie and we should get forwarded to Spotify's auth service
	r = e.GET("/").WithCookie(constants.ConsentCookieName, validConsentCookieValue()).Expect()
	r.Status(http.StatusTemporaryRedirect)
	r.Header("Location").Equal("https://auth.spotify.com/some-redirect-url")
}

func beforeEach(t *testing.T) (*httpexpect.Expect, *gomock.Controller, *mocks.MockPlayerStatesPersistor, *mocks.MockSpotAuthenticator, *mocks.MockSpotClient) {
	ctrl := gomock.NewController(t)

	daoMock := mocks.NewMockPlayerStatesPersistor(ctrl)
	authMock := mocks.NewMockSpotAuthenticator(ctrl)
	clientMock := mocks.NewMockSpotClient(ctrl)
	spotClientMockCreator := func(token *oauth2.Token) spotify.SpotClient {
		return clientMock
	}

	webRoot, err := filepath.Abs("../../")
	if err != nil {
		t.Fatalf("Could not get path of web root: %s", err)
	}

	handler := main.SetupForTest(daoMock, authMock, spotClientMockCreator, webRoot)

	e := httpexpect.WithConfig(httpexpect.Config{
		Client: &http.Client{
			Transport: httpexpect.NewBinder(handler),
			Jar:       httpexpect.NewJar(),
		},
		Reporter: httpexpect.NewAssertReporter(t),
		Printers: []httpexpect.Printer{
			httpexpect.NewDebugPrinter(t, true),
		},
	})

	return e, ctrl, daoMock, authMock, clientMock
}

func validConsentCookieValue() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}
