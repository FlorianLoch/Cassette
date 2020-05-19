default: run_local

.PHONY: prepare_webui webui clean deploy

run_local: compile webui start_heroko_local

prepare_webui:
	npm install

webui:
	grunt

compile:
	go install

start:
	spotistate

start_heroko_local:
	heroku local

clean:
	rm -rf webui

deploy:
	git push heroku master -f

docker:
	docker build . -t fdloch/spotistate

deploy_docker_heroku:
	heroku container:login
	heroku container:push web
	heroku container:release web
