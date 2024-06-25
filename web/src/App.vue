<template lang="pug">
#app.v-cloak
    .page-content
        .header.p-3.center-sm
            h1#title.only-mobile.align-bottom Cassette
                br
                small for Spotify
            img#logo.d-inline.mr-3(src="../assets/cassette_logo.svg")
            h1#title.not-on-mobile.d-inline.align-bottom Cassette
                | &nbsp;
                small for Spotify
        keep-alive
            router-view
    .footer.p-3.text-center
        p(v-if="$route.name === 'Main'")
            a(href="/?showHelp=true") Show help
        p Found an issue? Please create a ticket over at&nbsp;
            a(href="https://github.com/florianloch/cassette/issues" target="_blank") GitHub
            | &nbsp;or write to&nbsp;
            span.emailAddress
            |.
        p Content provided by Spotify&reg;.
            br
            | Check out Cassette's source code at
            |
            a(href="https://github.com/florianloch/cassette" target="_blank") GitHub
            | .
        p
            router-link(:to="{ name: 'Consent' }") Privacy policy / manage your data.
</template>
