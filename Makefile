default: run_local

.PHONY: webui-prepare webui clean deploy

run_local: compile webui
	make start

webui-prepare:
	npm install

webui:
	grunt

compile:
	go install

start:
	spotistate

deploy:
	git push heroku master -f

# docker-build: $(wildcard **/*.go) $(wildcard webui_src/**/*)
# .make/docker-build
docker-build:
	docker build . -t fdloch/spotistate
	mkdir -p .make/ && touch .make/docker-build

docker-run:
	docker run --env-file ./.env -p 8080:8080 fdloch/spotistate

deploy_docker_heroku:
	heroku container:login
	heroku container:push web
	heroku container:release web
