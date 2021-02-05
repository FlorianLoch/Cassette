default: cassette

cassette_bin = ./cassette
cov_profile = ./coverage.out
node_modules =  ./web/node_modules
web_dist = ./web/dist
package_json = ./web/package.json
all_go_files = $(shell find . -type f -name '*.go')
# Files inside './web/node_modules' and './web/dist' are not taken into account. Changes in the first should go
# along with changes in './web/package.json' which is considered.
all_web_files = $(shell find ./web -path $(node_modules) -prune -false -o -path $(web_dist) -prune -false -o -type f -name '*')
all_files = $(shell find . -path ./.make -prune -false -o -path $(node_modules) -prune -false -o -path $(web_dist) -prune -false -o -type f -name '*')

.PHONY: clean run test build-web docker-build docker-run heroku-deploy-docker heroku-init dokku-deploy coverage show-coverage

clean:
	rm -rf web/dist
	rm -rf .make
	rm cassette

run: ./web/dist/ ./cassette
	CASSETTE_NETWORK_INTERFACE=localhost ./cassette

test:
ifeq (, $(shell which richgo))
	go test ./...
else
	richgo test ./...
endif

coverage: ./coverage.out

./coverage.out: $(all_go_files)
	# This workaround of grepping together a list of packages which do not solely contain test code seems to
	# be not necesarry with go 1.15.7 anymore...
	# https://github.com/golang/go/issues/27333
	go test ./... -coverpkg=$(shell go list ./... | grep -v test | tr "\n" ",") -coverprofile=$(cov_profile)

show-coverage: $(cov_profile)
	go tool cover -html=$(cov_profile)

generate-mocks:
ifeq (, $(shell which mockgen))
	$(error "'mockgen' not found, consider installing it via 'go get github.com/golang/mock/mockgen'.")
endif
	mockgen \
		-source ./internal/spotify/abstractionLayer.go \
		-destination ./internal/e2e_test/mocks/spotifyMocks.go \
		-package "mocks"
	mockgen \
		-source ./internal/persistence/persistence.go \
		-destination ./internal/e2e_test/mocks/persistenceMocks.go \
		-package "mocks"

build-web: $(web_dist) $(node_modules)

# Check all files in web/ directory but IGNORE node_modules as this significantly slows down checking.
# In case the content of web/node_modules changes a call to clean is therefore required.
# The dir './web/node_modules' is added in order to force Make to first run 'yarn install' before trying to build the web app
$(web_dist): $(node_modules) $(all_web_files)
	yarn --cwd "./web" build

$(node_modules): $(package_json)
	yarn --cwd "./web" install

$(cassette_bin): $(all_go_files)
	go build .

docker-build: .make/docker-build

.make/docker-build: $(all_files)
	docker build . -t fdloch/cassette
	mkdir -p .make/ && touch .make/docker-build

docker-run: .make/docker-build
	docker run --env-file ./.env --env CASSETTE_PORT=8080 --env CASSETTE_NETWORK_INTERFACE=0.0.0.0 --env CASSETTE_APP_URL=http://192.168.108.176:8080 -p 8080:8080 fdloch/cassette

heroku-init: .make/heroku-login

heroku-deploy-docker: .make/heroku-login
	heroku container:login
	heroku container:push web
	heroku container:release web

.make/heroku-login:
	heroku login
	heroku git:remote -a audio-book-helper-for-spotify
	mkdir -p .make/ && touch .make/heroku-login

dokku-deploy: .make/dokku-deploy

.make/dokku-deploy: .make/docker-build
	docker tag fdloch/cassette:latest dokku/cassette:latest
	docker save dokku/cassette:latest | ssh florian@vps.fdlo.ch "docker load"
	ssh -t florian@vps.fdlo.ch "sudo dokku tags:deploy cassette latest"
	touch .make/dokku-deploy