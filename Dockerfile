# Version of golang image should be the same as used in Github CI
# We cannot use the alpine image anymore because we need to invoke `git` to fill the build args
FROM golang:1.22.4 AS gobuilder
WORKDIR /src/github.com/florianloch/cassette
# We run the next three lines before copying the workspace in order to avoid having Go download all modules everytime somethings changes
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN GOOS=linux GARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X main.gitVersion=$(git describe --always) -X main.gitAuthorDate=$(git log -1 --format=%aI) -X main.buildDate=$(date +%Y-%m-%dT%H:%M:%S%z)"


FROM node AS webbuilder
# According to https://nodejs.org/en/blog/release/v17.0.0/
ENV NODE_OPTIONS=--openssl-legacy-provider
WORKDIR /build
# We run the next three lines before copying ./web in order to avoid running 'yarn install' every time some file in ./web changes
COPY ./web/package.json .
COPY ./web/yarn.lock .
RUN yarn install

COPY ./web .
COPY .git/ .git/
RUN GIT_VERSION=$(git describe --always) GIT_AUTHOR_DATE=$(git log -1 --format=%aI) BUILD_DATE=$(date +%Y-%m-%dT%H:%M:%S%z) yarn build

FROM alpine
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY ./CHECKS .
COPY --from=gobuilder /src/github.com/florianloch/cassette/cassette .
COPY --from=webbuilder /build/dist ./web/dist
CMD ["./cassette"]
