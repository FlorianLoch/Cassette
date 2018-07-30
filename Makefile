default: run

.PHONY: prepare_webui webui run clean

prepare_webui:
	cd webui_src && npm install

webui:
	cd webui_src && grunt

run:
	go install && audioBookHelperForSpotify

clean:
	rm -rf webui