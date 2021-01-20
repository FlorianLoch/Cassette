(() => {
  const URL_CSRF_TOKEN = "/csrfToken";
  const CSRF_HEADER_NAME = "cassette_csrf_token";
  const URL_DATA = "/you";

  new Vue({
    el: '#app',
    data: {
    },
    methods: {
      fetchCSRFToken: function () {
        const self = this;
        return this.$http.head(URL_CSRF_TOKEN).then(res => {
          const csrfToken = res.headers.get(CSRF_HEADER_NAME)

          Vue.http.headers.common[CSRF_HEADER_NAME] = csrfToken
        }, res => {
          self.showErrorMessage("Could not fetch CSRF token!")
        })
      },
      giveConsent: function () {
        const now = encodeURIComponent(new Date().toUTCString())
        const maxAge = 10 * 60 * 60 * 24 * 365 // 10 years
        document.cookie = `cassette_consent=${now};max-age=${maxAge}`

        window.location.href = "/"
      },
      exportData: function () {
        window.location.href = URL_DATA
      },
      deleteData: function () {
        this.fetchCSRFToken().then(() => {
          return this.$http.delete(URL_DATA)
        }).then(() => {
          // TODO: Also delete all local cookies

          alert("Your data has successfully been removed from the system!")
        }, res => {
          alert("An error occurred. Did you already authorize Cassette via Spotify and actually save a state?")
          console.log(res)
        })
      }
    }
  })
})()