package e2e_test

import (
	"fmt"
	"net/http"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/golang/mock/gomock"
	"golang.org/x/oauth2"

	main "github.com/florianloch/cassette/internal"
	"github.com/florianloch/cassette/internal/constants"
	"github.com/florianloch/cassette/internal/e2e_test/mocks"
	"github.com/florianloch/cassette/internal/persistence"
	"github.com/florianloch/cassette/internal/spotify"
)

const (
	snippedFromIndexPage = "We're sorry but Cassette doesn't work properly without JavaScript enabled."
	spotifyAuthURL       = "https://auth.spotify.com/some-redirect-url"
	dummyUserID          = "audiophile_gopher"
)

var (
	dummyOAuthToken = &oauth2.Token{}
)

func TestConsentCheck(t *testing.T) {
	e, ctrl, _, authMock, _ := beforeEach(t)
	defer ctrl.Finish()

	authMock.EXPECT().AuthURL(gomock.Any()).Times(1).Return(spotifyAuthURL)

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
	cookieVal := validConsentCookieValue()
	r = e.GET("/").WithCookie(constants.ConsentCookieName, cookieVal).Expect()
	r.Status(http.StatusTemporaryRedirect)
	r.Header("Location").Equal(spotifyAuthURL)
	// Server should send back the consent cookie, this is helpful as we do not
	// have to attach it every time - httpexpect puts it into its cookie jar
	r.Cookie(constants.ConsentCookieName).Value().Equal(cookieVal)
}

func TestRetrievalOfPlayerStates(t *testing.T) {
	e, ctrl, daoMock, authMock, clientMock := beforeEach(t)
	defer ctrl.Finish()

	login(t, e, authMock)

	daoMock.EXPECT().LoadPlayerStates(dummyUserID).Times(0).
		Return([]*persistence.PlayerState{dummyPlayerState("book 1"), dummyPlayerState("book 2")}, nil)

	// currentUser gets stored in the session so should only be called once in the scope of a test
	clientMock.EXPECT().CurrentUser().Times(0)

	// TODO: Continue this!
}

func TestRetrievalOfActiveDevices(t *testing.T) {
	// TODO: implement!
}

func TestSavePlayerState(t *testing.T) {
	// TODO: implement!
	// 1. With invalid/not-attached CSRF token
	// 2. With specific slot
	// 3. Without specific slot
}

func TestRestorePlayerState(t *testing.T) {
	// TODO: implement!
	// 1. With specific device
	// 2. With default device
}

func TestDeletePlayerState(t *testing.T) {
	// TODO: implement!
}

func TestExportUserData(t *testing.T) {
	// TODO: implement!
}

func TestDeleteUserData(t *testing.T) {
	// TODO: implement!
}

func beforeEach(t *testing.T) (*httpexpect.Expect, *gomock.Controller, *mocks.MockPlayerStatesPersistor, *mocks.MockSpotAuthenticator, *mocks.MockSpotClient) {
	ctrl := gomock.NewController(t)

	daoMock := mocks.NewMockPlayerStatesPersistor(ctrl)
	authMock := mocks.NewMockSpotAuthenticator(ctrl)
	clientMock := mocks.NewMockSpotClient(ctrl)
	spotClientMockCreator := func(token *oauth2.Token) spotify.SpotClient {
		// Just for completeness and to check that the token is what we expect it to be
		// Can get called quite often, requests to almost any route cause a spotClient to be attached
		authMock.EXPECT().NewClient(dummyOAuthToken).AnyTimes()
		authMock.NewClient(token)

		return clientMock
	}

	webRoot, err := filepath.Abs("../../")
	if err != nil {
		t.Fatalf("Could not get path of web root: %s", err)
	}

	handler := main.SetupForTest(daoMock, authMock, spotClientMockCreator, webRoot)

	e := httpexpect.WithConfig(httpexpect.Config{
		BaseURL: "http://cassette.fdlo.ch",
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

func login(t *testing.T, e *httpexpect.Expect, authMock *mocks.MockSpotAuthenticator) {
	var givenState string
	authMock.EXPECT().AuthURL(gomock.Any()).Times(1).DoAndReturn(func(state string) string {
		if state == "" {
			t.Fatal("given state is empty")
		}

		givenState = state
		return fmt.Sprintf("%s?state=%s", spotifyAuthURL, state)
	})
	authMock.EXPECT().Token(newPointerMatcher(&givenState), gomock.Any()).Times(1).Return(dummyOAuthToken, nil)

	// 'initialRoute' is simply used to check whether the middleware remembers where we wanted to go
	// after having successfully authenticated
	r := e.GET("/initialRoute").WithCookie(constants.ConsentCookieName, validConsentCookieValue()).Expect()
	r.Status(http.StatusTemporaryRedirect)
	r.Header("Location").Equal(fmt.Sprintf("%s?state=%s", spotifyAuthURL, givenState))

	// We assume Spotify lets us in by simulating their callback
	// First with invalid state, then with valid one
	r = e.GET(constants.OAuthCallbackRoute).WithQuery("state", "bad State").Expect()
	r.Status(http.StatusBadRequest)
	r.Body().Contains("State mismatch in OAuth callback")

	// With the valid state we expect to be forwarded to the initially requested route...
	r = e.GET(constants.OAuthCallbackRoute).WithQuery("state", givenState).Expect()
	r.Status(http.StatusTemporaryRedirect)
	r.Header("Location").Equal("/initialRoute")

	// ... which should be the web app (SPA handler does not know 'initialRoute' and will serve default page)
	r = e.GET("/initialRoute").Expect()
	r.Status(http.StatusOK)
	r.Body().Contains(snippedFromIndexPage)
}

type pointerMatcher struct {
	ptr *string
}

func (p *pointerMatcher) Matches(x interface{}) bool {
	ptr := p.ptr
	return reflect.DeepEqual(*ptr, x)
}

func (p *pointerMatcher) String() string {
	ptr := p.ptr
	return fmt.Sprintf("is equal to %v", *ptr)
}

func newPointerMatcher(ptr *string) *pointerMatcher {
	return &pointerMatcher{ptr}
}

func dummyPlayerState(albumName string) *persistence.PlayerState {
	return &persistence.PlayerState{
		AlbumName: albumName,
	}
}
