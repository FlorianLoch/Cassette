<template lang="pug">
#app.v-cloak
  .page-content
    .header.p-3.center-sm
      h1#title.only-mobile.align-bottom Cassette
      img#logo.d-inline.mr-3(src="../assets/cassette_logo.svg")
      h1#title.not-on-mobile.d-inline.align-bottom Cassette
    keep-alive
      router-view
  .footer.p-3.text-center
    p
      | Content provided by Spotify&reg;.
      br
      | Check out Cassette's source code at
      |
      a(href="https://github.com/florianloch/cassette") GitHub
      | .
    p
      router-link(:to="{ name: 'Consent' }") Privacy policy / manage your data.
</template>

<script>
export default {
  name: "App",
  beforeCreate: () => {
    const gitVersion = process.env.GIT_VERSION
    const gitAuthorDate = process.env.GIT_AUTHOR_DATE
    const buildDate = process.env.BUILD_DATE

    console.info(`UI based on git commit: ${gitVersion}, authored at ${gitAuthorDate}, built at ${buildDate}`)
  }
}
</script>