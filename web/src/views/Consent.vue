<template lang="pug">
.container
  .jumbotron.mt-4
    h1.display-4 Welcome!
    p.lead Cassette for Spotify is a tool aiming at giving you the same comfort listening to audiobooks on Spotify&#174; your good, old cassette recorder provided while benefiting from Spotify's large collection and sublime portability. It does that by enabling you to pause the story you are listening to and to resume later without having to take screenshots or noting down the position every time.
    hr.my-4
    p But before we can start please read the following and give your consent. We take your privacy very serious and refrain from collecting/storing any data from you that is not stricly necessary in order to provide this service. We even take additional measures to anonymize you within our database. There is really nothing suprising going on, promised!
    p In the following this is explained in detail. In case of questions feel free to ask or to consult the source code of this application (see link at the bottom).
    p You can withdraw your consent at any time and may delete all data linked to you whenever you wish. You can also export it to have a look at the stored data linked to you. You find the controls further down this page. To get back here later please have a look at the bottom of the main web app once you gave your consent.
    p Your playing states are stored in a MongoDB database hosted by MongoDB Inc TODO: Link in Ireland.
    p There is an DPA (Data Processing Addendum) with Cloudflare, MongoDB incorporated data protection policies into their terms of service already. A DPA is therefore not required. TODO: What about Heroku? As you are a Spotify customer already and this service is solely acting on your behalf the terms of service and the privacy policy of Spotify apply to your (indirect) interactions with their platform.
    //- TODO: Do not show button when consent has been given already. Show date consent was given instead then. Do not show the links for exporting and deletion of data when consent has not been given yet.
    //- .row
    b-button(@click="giveConsent" variant="primary") Accept
    b-button(@click="exportData" variant="info") Export my data
    b-button(@click="deleteData" variant="danger") Delete my data
</template>

<script>
export default {
  name: "Consent",
  methods: {
    giveConsent: function () {
      const now = encodeURIComponent(new Date().toUTCString())
      const maxAge = 10 * 60 * 60 * 24 * 365 // 10 years
      document.cookie = `cassette_consent=${now};max-age=${maxAge}`

      window.location.href = "/"
    },
    exportData: function () {
      window.location.href = this.$api.URL_DATA
    },
    deleteData: function () {
      this.$api.fetchCSRFToken().then((csrfToken) => {
        this.$api.setCSRFToken(csrfToken)

        return this.$api.deleteYourData()
      }).then(() => {
        // TODO: Also delete all local cookies
        // TODO: Use modal instead of alert

        this.$bvModal.msgBoxOk("Your data has successfully been removed from the system!")
      }, (err) => {
        this.$bvModal.msgBoxOk("An error occurred. Did you already authorize Cassette via Spotify and actually save a state?")

        console.error("Failed deleting user data.", err)
      })
    }
  }
}
</script>


