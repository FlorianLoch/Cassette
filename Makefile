default: run_local

.PHONY: web_dist-prepare web_dist clean deploy

run_local: install web_dist
	spotistate

web_dist-prepare:
	npm install

web_dist:
	grunt

install:
	go install

docker-build:
	docker build . -t fdloch/spotistate
	mkdir -p .make/ && touch .make/docker-build

docker-run:
	docker run --env-file ./.env -p 8080:8080 fdloch/spotistate

deploy_docker_heroku:
	heroku container:login
	heroku container:push web
	heroku container:release web

heroku_init:
	heroku login
	heroku git:remote -a audio-book-helper-for-spotify