default: cassette

.PHONY: run clean deploy build-web

clean:
	rm -rf web/dist
	rm cassette

run: web/dist/ cassette
	./cassette

build-web: web/dist/

# Check all files in web/ directory but IGNORE node_modules as this significantly slows down checking.
# In case the content of web/node_modules changes a call to clean is therefore required.
web/dist/: $(shell find ./web  -path ./web/node_modules -prune -false -o -type f -name '*.*')
	yarn --cwd "./web" build

cassette: $(shell find ./ -type f -name '*.go')
	go build .

docker-build:
	docker build . -t fdloch/cassette
	mkdir -p .make/ && touch .make/docker-build

docker-run:
	docker run --env-file ./.env --env CASSETTE_PORT=8080 --env CASSETTE_NETWORK_INTERFACE=0.0.0.0 -p 8080:8080 fdloch/cassette

deploy-docker-heroku:
	heroku container:login
	heroku container:push web
	heroku container:release web

heroku-init:
	heroku login
	heroku git:remote -a audio-book-helper-for-spotify