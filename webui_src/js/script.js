(() => {
  Vue.http.interceptors.push((req, nxt) => {
    NProgress.start();
    nxt(() => {
        NProgress.done();
    });
  });

  const URL_CSRF_TOKEN = "/csrfToken";
  const URL_PLAYER_STATES = "/playerStates";
  const CSRF_HEADER_NAME = "X-CSRF-Token";

  // TODO Add error handlers

  const app = new Vue({
    el: '#app',
    data: {
      message: 'Hello Vue!',
      playerStates: []
    },
    methods: {
      fetchCSRFToken: function (done) {
        this.$http.head(URL_CSRF_TOKEN).then(res => {
          csrfToken = res.headers.get(CSRF_HEADER_NAME);

          Vue.http.headers.common[CSRF_HEADER_NAME] = csrfToken;

          done();
        }, res => {
          console.error("Could not fetch CSRF token!");
        });
      },
      fetchPlayerStates: function () {
        const self = this;
        this.$http.get(URL_PLAYER_STATES).then(res => {
          self.playerStates = res.body.states;
        }, res => {
          console.error("Requesting player states from backend failed!", res)
        });
      },
      storePlayerStateInSlot: function (slotNo) {
        const self = this;
        this.$http.put(`${URL_PLAYER_STATES}/${slotNo}`).then(res => {
          console.info("Successfully stored player state in slot!");
          self.fetchPlayerStates();
        }, res => {
          console.error(`Requesting to persist current player state in slot (${slotNo}) failed!`, res)
        });
      },
      storePlayerStateInNewSlot: function () {
        const self = this;
        this.$http.post(URL_PLAYER_STATES).then(res => {
          console.info("Successfully stored player state in new slot!");
          self.fetchPlayerStates();
        }, res => {
          console.error("Requesting to persist current player state failed!", res)
        });
      },
      removeSlot: function (slotNo) {
        const self = this;
        this.$http.delete(`${URL_PLAYER_STATES}/${slotNo}`).then(res => {
          console.info("Successfully removed slot!");
          self.playerStates.splice(slotNo, 1);
        }, res => {
          console.error(`Requesting to delete slot (${slotNo}) failed!`, res)
        });
      },
      restorePlayerStateFromSlot: function (slotNo) {
        this.$http.post(`${URL_PLAYER_STATES}/${slotNo}/restore`).then(res => {
          console.info("Successfully restored player state!");
        }, res => {
          console.error(`Requesting to restore player state from slot (${slotNo}) failed!`, res)
        });
      }
    },
    mounted: function () {
      this.fetchCSRFToken(this.fetchPlayerStates);
    }
  });
})();
