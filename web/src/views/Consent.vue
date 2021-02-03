<template lang="pug">
.container
  .jumbotron.mt-4
    h1.display-4 Welcome!
    p.lead Cassette for Spotify is a tool aiming at giving you the same comfort listening to audiobooks on Spotify&#174; your good, old cassette recorder provided while benefiting from Spotify's large collection and sublime portability. It does that by enabling you to pause the story you are listening to and to resume later without having to take screenshots or noting down the position every time.
    hr.my-4
    p But before we can start please read the following and give your consent. We take your privacy very serious and refrain from collecting/storing any data from you that is not stricly necessary in order to provide this service. We even take additional measures to anonymize you within our database. There is really nothing suprising going on, promised!
    p In a nutshell: You grant Spotify to grant his service access to your player state. Cassette reads and writes this state as you request it to do so. The token enabling Cassette to perform these operations is stored in your browser. Your states are stored in a database hosted by MongoDB. There are no operations performed using your data except the ones stated. Currently there is no tracking, advertisement or the like within this webapp. You can withdraw your consent at any time. As always this software is offered as-is, it comes with no more than the minimum liability required by the applicable laws.
    .row.mx-auto.mb-4
      b-button(@click="giveConsent" variant="primary" :disabled="$api.consentGiven()") Accept

    p In more detail: You will be forwarded to Spotify's login service and will be asked whether to grant Cassette access to your profile (this is mandatory, we need to access the player state). Spotify will then issue a token to Cassette enabling it to access your player state. As this token is confidential it will only be processed on our systems, it will not be stored there but only inside an encrypted cookie within your browser. We have no access to your accounts password etc., this token can only be used to perform the actions you granted us when being asked by Spotify. Your player states are stored in a hosted database with your user name (also refered to as ID) being anonymised. As this data is your data with need you to accept us handling it as described on this page. We do not analyze your taste in music nor trace your behavior, we solely need it to request your current player state from spotify, to link it with you in our database and to restore them later. In case of questions please read on. Also feel free to ask or to consult the source code of this application (see link at the bottom).

    p Your session data &ndash; mainly the token issued to us by Spotify on your behalf granting us access to your player states and user id &ndash; is not persisted on the server. It is stored in an encrypted cookie living inside your browser. The name of this cookie is "cassette_session". In order to not display you this consent page everytime we store your decision in "cassette_consent" (only in case you give consent). Additionally there is a cookie named "cassette_csrf" being required for technical reasons (i.e. to prevent CSRF attacks).
    p In order to provide this service Cassette uses some third-party service providers:
    ul
      li Netcup: The application is running on a server hosted by Netcup. It is a German company oblidged to German data privacy laws.
      li MongoDB: Your player states are stored in databases hosted by them (in Ireland). Your user name (Spotify user ID) gets anonymised before being written to the database. Anonymisation happens by hashing your user ID, this operation is considered to be irreversible. But having access to the database and knowing your user ID one could prove it is in there. MongoDB incorporated data protection policies into their terms of service already, an additional Data Protection Addendum (DPA) therefore is not necessary.
      li Cloudflare: Their service is used to provide HTTPS encryption, DDoS protection and caching. They comply with the GDPR; a DPA has been signed with them.
      li Spotify: This service depends on the API provided by Spotify. As you need to be a Spotify customer already in order to use this tool and this service is solely acting on your behalf the terms of service and the privacy policy of Spotify apply to your (indirect) interactions with their platform.
    p You can withdraw your consent at any time and may delete all data linked to you whenever you wish. You can also export it to have a look at the stored data linked to you. You find the controls below. To get back here later please have a look at the bottom of the main web app once you gave your consent.
    .row.mx-auto
      //- TODO: Add a button for revoking consent and display this when consent has been given. Also delete all cookies then.
      b-button(@click="giveConsent" variant="primary" :disabled="$api.consentGiven()") Accept
      b-button.ml-1(@click="exportData" variant="info") Export my data
      b-button.ml-1(@click="deleteData" variant="danger") Delete my data / Withdraw my consent
</template>

<script>
export default {
  name: "Consent",
  methods: {
    giveConsent: function () {
      this.$api.giveConsent()

      // We have to explicitly trigger the browser to reload the page in order
      // for the Spotify OAuth redirect to kick in.
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
        this.$api.withdrawConsent()

        this.$bvModal.msgBoxOk("Your data has successfully been removed from the database! Due to technical reasons we can not enforce deletion of the data we stored in your browser. Please delete these cookies ('cassette_session' and 'cassette_csrf') manually resp. using your browser's tools.")
      }, (err) => {
        this.$bvModal.msgBoxOk("An error occurred. Did you already authorize Cassette via Spotify and actually save a state?")

        console.error("Failed deleting user data.", err)
      })
    }
  }
}
</script>


