module github.com/florianloch/cassette

go 1.15

require (
	github.com/NYTimes/gziphandler v1.1.1
	github.com/gavv/httpexpect/v2 v2.2.0 // lock to this version, test breaks when upgrading
	github.com/go-chi/chi v1.5.4
	github.com/golang/mock v1.6.0
	github.com/gorilla/csrf v1.7.1
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/sessions v1.2.1
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/rs/zerolog v1.26.1
	github.com/zmb3/spotify v1.3.0
	go.mongodb.org/mongo-driver v1.8.3
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
