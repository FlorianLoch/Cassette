<template lang="pug">
.container
  .jumbotron.mt-4
    h1.display-4 Welcome!
    p.lead Cassette for Spotify is a tool trying to give you the same comfort listening to audiobooks on Spotify&#174; as your good, old cassette recorder provided while benefitting from Spotify's large collection and sublime portability. It does that by enabling you to suspend the story you are listening to and resume later without having to take screenshots, note down or simply remember the position every time.
    hr.my-4
    p But before we can start please read the following and give your consent. Do not be afraid of this lengthy text &ndash; but data protection is important to us and we want to clarify how your data is used within Cassette. We take your privacy very serious and refrain from collecting/storing any data from you that is not strictly necessary in order to provide this service. We even take additional measures to anonymize you within our database. There is really nothing surprising going on, promised!
    p In a nutshell: You grant Spotify to grant this service access to your "player state". Cassette reads and writes this state as you request it to do so. The token enabling Cassette to perform these operations is stored in your browser. Your states are stored in a database hosted by a company called MongoDB. There are no operations performed using your data except the ones stated. Currently there is no tracking, advertisement or the like within this web app. You can withdraw your consent at any time. As always, this software is offered as-is &ndash; it comes with no more than the minimum liability required by the applicable laws.
    .row.mx-auto.mb-4
      template(v-if="$api.consentGiven()")
        b-button(@click="goToApp", variant="primary") Go back to the app, you already gave your consent
      template(v-else)
        b-button(@click="giveConsent", variant="primary") Accept

    p In more detail: You will be forwarded to Spotify's login service and will be asked whether to grant Cassette access to your profile (this is mandatory, we need to access the player state). Spotify will then issue a token to Cassette enabling it to access your player state. As this token is confidential it will only be processed on our systems, it will not be stored anywhere but only inside an encrypted cookie within your browser. We have no access to your account's password etc. This token can only be used to perform the actions you granted Cassette when being asked by Spotify. Your player states are stored in a hosted database with your user name (also referred to as "ID") being anonymized. As this data is your data we need you to accept us handling it as described on this page. We do not analyze your taste in music nor trace your behavior &ndash; we solely need it to request your current player state from Spotify, to link it with you in our database, to restore states later and to request your active devices (in order to provide you with the option to choose on which device you want to resume). In case of questions please read on. Also feel free to ask or to consult the source code of this application (see link at the bottom).

    p Your session data &ndash; mainly the token issued to us by Spotify on your behalf granting us access to your player states and user ID &ndash; are not persisted on the server. It is stored in an encrypted cookie stored inside your browser. The name of this cookie is "cassette_session". In order to not display you this consent page everytime we store your decision in "cassette_consent" (only in case you give consent, of course). Additionally there is a cookie named "cassette_csrf" being required for technical reasons (i.e. to prevent CSRF attacks). This information is, at max, stored as long as you use this service, resp. until you request deletion (see below).
    p In order to provide this service Cassette uses some third-party service providers:
    ul
      li Netcup&#174;: The application is running on a server hosted by Netcup. It is a German company obliged to German data privacy laws.
      li MongoDB&#174;: Your player states are stored in databases hosted by them (on Azure servers located in the Netherlands). Your user name (Spotify user ID) gets anonymized before being written to the database. Anonymization happens by hashing your user ID. This operation is considered to be irreversible. But having access to the database and knowing your user ID one could find your record in it. MongoDB incorporated data protection policies into their terms of service already, an additional Data Protection Addendum (DPA) therefore is not necessary.
      li Cloudflare&#174;: Their service is used to provide HTTPS encryption, DDoS protection and caching. They comply with the GDPR; a DPA has been signed with them.
      li Spotify&#174;: This service depends on the API provided by Spotify. As you need to be a Spotify customer already in order to use this tool and this service is solely acting on your behalf the terms of service and the privacy policy of Spotify apply to your (indirect) interactions with their platform. All content, like album names and cover arts, is provided by Spotify. You agree to only use this content as permitted to you by Spotify.
    p You can withdraw your consent at any time and may delete all data linked to you whenever you wish. You can also export it to have a look at the stored data linked to you. You can find the controls below. To get back here later please have a look at the bottom of the main web app once you gave your consent.
    p This service is provided by: Florian Loch (
      span#emailAddress
      | ).
    p Feel free to ask your questions about Cassette. Please report bugs and abuse.

    .row.mx-auto
      template(v-if="$api.consentGiven()")
        b-button(@click="goToApp", variant="primary") Go back to the app, you already gave your consent
      template(v-else)
        b-button(@click="giveConsent", variant="primary") Accept
      b-button.ml-1(@click="exportData", variant="info") Export my data
      b-button.ml-1(@click="deleteData", variant="danger") Delete my data &amp; withdraw my consent
</template>

<script>
export default {
  name: "Consent",
  methods: {
    goToApp: function () {
      this.$router.push({ name: "Main" })
    },
    giveConsent: function () {
      this.$api.giveConsent()

      // We have to explicitly trigger the browser to reload the page in order
      // for the Spotify OAuth redirect to kick in.
      location.assign(`/?nocache=${new Date().getTime()}&showRun=true`);
    },
    exportData: function () {
      location.assign(this.$api.URL_DATA)
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


