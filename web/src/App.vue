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