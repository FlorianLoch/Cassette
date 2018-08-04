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

run_heroku:
	audioBookHelperForSpotify ${PORT}

clean:
	rm -rf webui