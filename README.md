# Cassette

![](./cassette_logo.svg)

Cassette is a small web application enabling you to pause and resume audiobooks on Spotify. Spotify offers a lot of great audiobooks but as its built with music in a broader sense in mind it does not provide essential features required in order to comfortably listen to them: pausing an audiobook and listening to some music results in you needing to think about in which chapter you left off.

The common workaround is noting your progress down manually, e.g., as a text note or a screenshot. But there are severeal reasons why this is uncomfortable.

Cassette tries to overcome this problem by enabling you to store your state, put is aside and resume it later. Like you would have done it with a cassette in the old times.


## How is this done?
Simply spoken by using the Spotify Web API. Cassette itself consists of two components, a REST service running on a server (this directory) and a web app (./web) running in your browser. The service is talking with the Spotify Web API and a MongoDB database in with the states get persisted. The web app talks with the service via a REST interface.


## Current status of the project
There has been a first version, basically a proof-of-concept for quite some time. I use it quite often and by the time I considered it quite useful and decided to rewrite the project in a more thorough fashion with the goal of making the tool available to everyone who wants to use it. Admittedly, this is also a play project for trying out stuff and a "finger exercise". ;)

I am aware that, at its current state this project does not necessarily meets best standards in software development, e.g., there are no tests and some of the code is still in proof-of-concept state. But this will hopefully change in the near future.

### Prototype
Version 1 is currently deployed at Heroku (as a Docker container) and can be found at [spotistate.fdlo.ch](https://spotistate.fdlo.ch). Note that the prototype does not reflect the current state of this repository and still uses the old name, which could not be kept as it violates Spotify's branding guidelines.

## Disclaimer
The authors of this project are not related to Spotify in any way beside being happy users of their platform. This service is not related to Spotify except using their API and content.