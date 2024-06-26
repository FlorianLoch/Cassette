default: build-all

.PHONY: build-all clean run-server run test build-web docker-build docker-run heroku-deploy-docker heroku-init dokku-deploy coverage show-coverage lint install-hooks

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

git_version = $(shell git describe --always)
git_author_date = $(shell git log -1 --format=%aI)
build_date = $(shell date +%Y-%m-%dT%H:%M:%S%z)

# Load the env file and export its variable to all programms invoked by make
-include .env # - suppresses the error in case the file does not exist
export

install-hooks:
	rm -f .git/hooks/pre-commit
	ln -s ../../pre-commit.sh .git/hooks/pre-commit

build-all: $(web_dist) $(cassette_bin)

clean:
	rm -rf $(web_dist)
	rm -rf .make
	rm $(cov_profile) || true
	rm $(cassette_bin) || true

run-server: $(cassette_bin)
	CASSETTE_NETWORK_INTERFACE=localhost $(cassette_bin)

run: build-all
	CASSETTE_NETWORK_INTERFACE=localhost $(cassette_bin)

lint: .make/go-lint

.make/go-lint: $(cassette_bin)
	golangci-lint run ./...
	mkdir -p .make/ && touch .make/go-lint

test: $(web_dist) # web dist needs to be available for testing
ifeq (, $(shell which richgo))
	go test ./...
else
	richgo test ./...
endif

coverage: $(cov_profile)

$(cov_profile): $(all_go_files)
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
	cd ./web; GIT_VERSION=$(git_version) GIT_AUTHOR_DATE=$(git_author_date) BUILD_DATE=$(build_date) yarn build
	touch $(web_dist)

$(node_modules): $(package_json)
	cd ./web; yarn install
	touch $(node_modules)

$(cassette_bin): $(all_go_files)
	go build -ldflags "-X main.gitVersion=$(git_version) -X main.gitAuthorDate=$(git_author_date) -X main.buildDate=$(build_date)"

docker-build: .make/docker-build

.make/docker-build: $(all_files)
	docker build -t fdloch/cassette .
	mkdir -p .make/ && touch .make/docker-build

docker-run: .make/docker-build
	docker run --env-file ./.env --env CASSETTE_PORT=8082 --env CASSETTE_NETWORK_INTERFACE=0.0.0.0 --env CASSETTE_APP_URL=http://192.168.108.176:8082 -p 8082:8082 fdloch/cassette

dokku-deploy: .make/dokku-deploy

.make/dokku-deploy: test $(all_files)
	-git remote add dokku dokku@vps.fdlo.ch:cassette
	git push --force dokku
	touch .make/dokku-deploy
