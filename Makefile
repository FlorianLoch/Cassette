default: run

.PHONY: prepare_webui webui run clean

prepare_webui:
	npm install

webui:
	grunt

run: webui compile start

compile:
	go install

start:
	audioBookHelperForSpotify

start_heroko_local:
	heroku local

run_local: compile webui start_heroko_local

clean:
	rm -rf webui