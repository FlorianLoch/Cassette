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
	spotifyAPI "github.com/zmb3/spotify"
	"golang.org/x/oauth2"

	main "github.com/florianloch/cassette/internal"
	"github.com/florianloch/cassette/internal/constants"
	"github.com/florianloch/cassette/internal/e2e_test/mocks"
	"github.com/florianloch/cassette/internal/persistence"
	"github.com/florianloch/cassette/internal/spotify"
)

const (
	snippetFromIndexPage = "We're sorry but Cassette doesn't work properly without JavaScript enabled."
	spotifyAuthURL       = "https://auth.spotify.com/some-redirect-url"
	dummyUserID          = "audiophile_gopher"
)

var (
	dummyOAuthToken = &oauth2.Token{}
	dummyUser       = &spotifyAPI.PrivateUser{
		User: spotifyAPI.User{
			ID: dummyUserID,
		},
	}
	dummyDevices = []spotifyAPI.PlayerDevice{{
		ID:   "001",
		Name: "Device 1",
	}, {
		ID:     "002",
		Name:   "Device 2",
		Active: true,
	}}
)

func TestAPIHasOwn404Route(t *testing.T) {
	// Routes not hanging below the "/api" node are handled by the SPA middleware
	e, _, _, _, _ := beforeEach(t)

	r := e.GET("/api/activeDevices").Expect()
	r.Status(http.StatusForbidden)

	r = e.GET("/api/currentDevices").Expect()
	r.Status(http.StatusNotFound)
}

func TestConsentCheck(t *testing.T) {
	e, ctrl, _, authMock, _ := beforeEach(t)
	defer ctrl.Finish()

	authMock.EXPECT().AuthURL(gomock.Any()).Times(1).Return(spotifyAuthURL)

	// Requesting without consent token API routes should be reachable
	r := e.HEAD("/api/csrfToken").Expect()
	r.Header(constants.CSRFHeaderName).NotEmpty()
	r.Cookie(constants.CSRFCookieName).Value().NotEmpty()

	// Requests for index will always be served the web app until we provide a valid consent cookie
	e.GET("/").Expect().Body().Contains(snippetFromIndexPage)

	// Check whether an invalid consent cookie gets handled correctly
	r = e.GET("/").WithCookie(constants.ConsentCookieName, "ancient cookie, tastes really bad").Expect()
	r.Header(constants.ConsentNoticeHeaderName).IsEqual("ATTENTION: consent not given yet.")
	r.Body().Contains(snippetFromIndexPage)

	// Now try again with a valid cookie; we should get forwarded to Spotify's auth service
	cookieVal := validConsentCookieValue()
	r = e.GET("/").
		WithRedirectPolicy(httpexpect.DontFollowRedirects).
		WithCookie(constants.ConsentCookieName, cookieVal).
		Expect()
	r.Status(http.StatusTemporaryRedirect)
	r.Header("Location").IsEqual(spotifyAuthURL)
	// Server should send back the consent cookie, this is helpful as we do not
	// have to attach it every time - httpexpect puts it into its cookie jar
	r.Cookie(constants.ConsentCookieName).Value().IsEqual(cookieVal)
}

func TestRetrievalOfPlayerStates(t *testing.T) {
	e, ctrl, daoMock, authMock, clientMock := beforeEach(t)
	defer ctrl.Finish()

	login(t, e, authMock)

	daoMock.EXPECT().LoadPlayerStates(dummyUserID).Times(1).
		Return([]*persistence.PlayerState{dummyPlayerState("book 1"), dummyPlayerState("book 2")}, nil)

	// currentUser gets stored in the session so should only be called once in the scope of a test
	clientMock.EXPECT().CurrentUser().Times(1).Return(dummyUser, nil)

	r := e.GET("/api/playerStates").Expect()
	r.Status(http.StatusOK)
	r.HasContentType("application/json")
	a := r.JSON().Array()
	a.Length().IsEqual(2)
	a.Value(0).Object().Value("albumName").String().IsEqual("book 1")
	a.Value(1).Object().Value("albumName").String().IsEqual("book 2")
}

func TestRetrievalOfActiveDevices(t *testing.T) {
	e, ctrl, _, authMock, clientMock := beforeEach(t)
	defer ctrl.Finish()

	login(t, e, authMock)

	clientMock.EXPECT().PlayerDevices().Times(1).Return(dummyDevices, nil)

	r := e.GET("/api/activeDevices").Expect()
	r.Status(http.StatusOK)
	r.HasContentType("application/json")
	a := r.JSON().Array()
	a.Length().IsEqual(2)
	o1 := a.Value(0).Object()
	o1.Value("name").String().IsEqual("Device 1")
	o1.Value("active").Boolean().IsFalse()
	o2 := a.Value(1).Object()
	o2.Value("name").String().IsEqual("Device 2")
	o2.Value("active").Boolean().IsTrue()
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
		BaseURL: "http://cassette-for-spotify.app",
		Client: &http.Client{
			Transport: httpexpect.NewBinder(handler),
			Jar:       httpexpect.NewCookieJar(),
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
	r := e.GET("/initialRoute?someParameter=someValue#someAnchor").
		WithRedirectPolicy(httpexpect.DontFollowRedirects).
		WithCookie(constants.ConsentCookieName, validConsentCookieValue()).
		Expect()
	r.Status(http.StatusTemporaryRedirect)
	r.Header("Location").IsEqual(fmt.Sprintf("%s?state=%s", spotifyAuthURL, givenState))

	// We assume Spotify lets us in by simulating their callback
	// First with invalid state, then with valid one
	r = e.GET(constants.OAuthCallbackRoute).WithQuery("state", "bad State").Expect()
	r.Status(http.StatusBadRequest)
	r.Body().Contains("State mismatch in OAuth callback")

	// With the valid state, we expect to be forwarded to the initially requested route...
	r = e.GET(constants.OAuthCallbackRoute).WithRedirectPolicy(httpexpect.DontFollowRedirects).WithQuery("state", givenState).Expect()
	r.Status(http.StatusTemporaryRedirect)
	r.Header("Location").IsEqual("/initialRoute?someParameter=someValue#someAnchor")

	// ... which should be the web app (SPA handler does not know 'initialRoute' and will serve default page)
	r = e.GET("/initialRoute").Expect()
	r.Status(http.StatusOK)
	r.Body().Contains(snippetFromIndexPage)

	// Requesting the callback route again, we should be forwarded to the entry page again
	// because there is a token within our session
	r = e.GET(constants.OAuthCallbackRoute).WithRedirectPolicy(httpexpect.DontFollowRedirects).Expect()
	r.Status(http.StatusTemporaryRedirect)
	r.Header("Location").IsEqual("/")
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
