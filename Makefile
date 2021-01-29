default: cassette

.PHONY: clean run test build-web docker-build docker-run heroku-deploy-docker heroku-init dokku-deploy

clean:
	rm -rf web/dist
	rm -rf .make
	rm cassette

run: ./web/dist/ ./cassette
	./cassette

test:
	go test ./...

build-web: ./web/dist/

# Check all files in web/ directory but IGNORE node_modules as this significantly slows down checking.
# In case the content of web/node_modules changes a call to clean is therefore required.
./web/dist/: $(shell find ./web -path ./web/node_modules -prune -false -o -path ./web/dist -prune -false -o -type f -name '*')
	yarn --cwd "./web" build

./cassette: $(shell find ./ -type f -name '*.go')
	go build .

docker-build: .make/docker-build

.make/docker-build: $(shell find . -path ./web/node_modules -prune -false -o -not -path '*/\.*' -type f -name '*')
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