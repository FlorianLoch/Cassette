<template lang="pug">
  #app.v-cloak
    .header
      h3
        | Cassette
        small.text-muted.ml-2 for Spotify
    keep-alive
      router-view
    .footer
      | Powered by
      |
      a(href="https://golang.org" target="_blank") Go
      | ,
      |
      a(href="https://vuejs.org" target="_blank") Vue.js
      |
      | and
      |
      a(href="https://getbootstrap.com" target="_blank") Bootstrap
      |. Favicon (edited) and other icons originate from the
      |
      a(href="https://fontawesome.com" target="_blank") Font Awesome project
      |
      | (see
      |
      a(href="https://fontawesome.com/license" target="_blank") license
      | ).
      br
      | Check out at
      |
      a(href="https://github.com/florianloch/cassette") Github
      |.
      //- TODO: Include this again
      //- br
      //- | Running UI version #{version}, built at #{buildDate}.
</template>

