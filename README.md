![build and test](https://github.com/FlorianLoch/Cassette/actions/workflows/ci_cd.yaml/badge.svg)
[![Coverage Status](https://coveralls.io/repos/github/FlorianLoch/Cassette/badge.svg?branch=master)](https://coveralls.io/github/FlorianLoch/Cassette?branch=master)

# Cassette

![](./web/assets/cassette_logo.svg)

An "audio book helper utility for Spotify&reg;".

***Try it out at https://cassette-for-spotify.app.***

Cassette is a small web application enabling you to pause and resume audiobooks on Spotify. Spotify offers a lot of great audiobooks but as its built with music in a broader sense in mind it does not provide essential features required in order to comfortably listen to them: pausing an audiobook and listening to some music results in you needing to think about in which chapter you left off.

The common workaround is noting your progress down manually, e.g., as a text note or a screenshot. But there are several reasons why this is uncomfortable.

Cassette tries to overcome this problem by enabling you to store your state, put is aside and resume it later. Like you would have done it with a cassette in the old times.


## How is this done?
In a nutshell: By using the Spotify Web API. 

Cassette itself consists of two parts: 
- a REST service running on a server (this directory) 
- a web app (./web) running in your browser

The service is talking with the Spotify Web API, and a MongoDB database in with the states get persisted. 
The web app talks with the service via a REST interface.


## Current status of the project
After spending a lot of time rewriting all parts of this project, I finally was able to release version 2. 
Although version 1, which had another name that could not be kept as it violated Spotify's branding guidelines, was available to the public already, it was not really too robust, not really intuitive to use and overall not meant to be used by anyone but myself. 
It was more a proof-of-concept project hacked together in a few days.

Version 2 is way more mature, takes care of data privacy, includes an interactive introduction tour and a polished UI with both mobile and desktop users in mind. 
Its goal is to be, hopefully, useful to many users aside myself.

Version 2 is deployed at https://cassette-for-spotify.app.


## Disclaimer
The authors of this project are not related to Spotify in any way besides being happy users of their platform. 
This service is not related to Spotify; it is only using their API and content.