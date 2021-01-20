(() => {
  NProgress.configure({ showSpinner: false });

  Vue.http.interceptors.push(() => {
    NProgress.start();
    return () => {
      NProgress.done();
    };
  });

  Vue.component('modal', {
    template: '#modal-template'
  });

  const URL_CSRF_TOKEN = "/csrfToken"
  const CSRF_HEADER_NAME = "cassette_csrf_token"
  const URL_PLAYER_STATES = "/playerStates"
  const URL_ACTIVE_DEVICES = "/activeDevices"

  // TODO Add error handlers

  new Vue({
    el: '#app',
    data: {
      playerStates: [],
      activeDevices: [],
      showModal: false,
      errorMessage: ""
    },
    filters: {
      time: function (millis) {
        const inSecs = Math.round(millis / 1000)
        const hours = Math.floor(inSecs / 3600)
        const remaining = inSecs - hours * 3600
        const minutes = Math.floor(remaining / 60)
        const seconds = remaining - minutes * 60

        return `${(hours > 0) ? hours + ":" : ""}${(minutes < 10) ? "0" : ""}${minutes}:${(seconds < 10) ? "0" : ""}${seconds}`
      }
    },
    methods: {
      showErrorMessage: function (msg) {
        this.errorMessage = msg;
        this.showModal = true;
      },
      fetchCSRFToken: function (done) {
        const self = this;
        this.$http.head(URL_CSRF_TOKEN).then(res => {
          const csrfToken = res.headers.get(CSRF_HEADER_NAME);

          Vue.http.headers.common[CSRF_HEADER_NAME] = csrfToken;

          done();
        }, res => {
          self.showErrorMessage("Could not fetch CSRF token!");
        });
      },
      fetchActiveDevices: function () {
        const self = this;
        this.$http.get(URL_ACTIVE_DEVICES).then(res => {
          self.activeDevices = res.body;
          console.log(res.body);
        }, res => {
          self.showErrorMessage("Requesting active devices from backend failed: " + res.body);
        });
      },
      fetchPlayerStates: function () {
        const self = this;
        this.$http.get(URL_PLAYER_STATES).then(res => {
          self.playerStates = res.body.states;
        }, res => {
          self.showErrorMessage("Requesting player states from backend failed: " + res);
        });
      },
      storePlayerStateInSlot: function (slotNo) {
        const self = this;
        this.$http.put(`${URL_PLAYER_STATES}/${slotNo}`).then(res => {
          console.info("Successfully stored player state in slot!");
          self.fetchPlayerStates();
        }, res => {
          self.showErrorMessage(`Requesting to persist current player state in slot (${slotNo}) failed: ` + res.body);
        });
      },
      storePlayerStateInNewSlot: function () {
        const self = this;
        this.$http.post(URL_PLAYER_STATES).then(res => {
          console.info("Successfully stored player state in new slot!");
          self.fetchPlayerStates();
        }, res => {
          self.showErrorMessage("Requesting to persist current player state failed: " + res.body);
        });
      },
      removeSlot: function (slotNo) {
        const self = this;
        this.$http.delete(`${URL_PLAYER_STATES}/${slotNo}`).then(res => {
          console.info("Successfully removed slot!");
          self.playerStates.splice(slotNo, 1);
        }, res => {
          self.showErrorMessage(`Requesting to delete slot (${slotNo}) failed: ` + res.body);
        });
      },
      restorePlayerStateFromSlot: function (slotNo, deviceID) {
        const url = `${URL_PLAYER_STATES}/${slotNo}/restore${(deviceID) ? `?deviceID=${deviceID}` : ""}`;
        const self = this;
        this.$http.post(url).then(res => {
          console.info("Successfully restored player state!");
        }, res => {
          self.showErrorMessage(`Requesting to restore player state from slot (${slotNo}) failed: ` + res.body);
        });
      }
    },
    mounted: function () {
      this.fetchActiveDevices();
      this.fetchCSRFToken(this.fetchPlayerStates);
    }
  });
})();

