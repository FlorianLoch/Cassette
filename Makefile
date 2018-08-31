default: run_local

.PHONY: prepare_webui webui clean

run_local: compile webui start_heroko_local

prepare_webui:
	npm install

webui:
	grunt

compile:
	go install

start:
	audioBookHelperForSpotify

start_heroko_local:
	heroku local

clean:
	rm -rf webui