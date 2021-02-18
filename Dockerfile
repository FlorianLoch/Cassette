# Version of golang image should be the same as used in Github CI
FROM golang:1.15.7-alpine AS gobuilder
ARG GIT_VERSION
ARG GIT_AUTHOR_DATE
ARG BUILD_DATE
WORKDIR /src/github.com/florianloch/cassette
# We run the next three lines before copying the workspace in order to avoid having Go download all modules everytime somethings changes
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN GOOS=linux GARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X main.gitVersion=$GIT_VERSION -X main.gitAuthorDate=$GIT_AUTHOR_DATE -X main.buildDate=$BUILD_DATE"


FROM node AS web_distbuilder
WORKDIR /build
# We run the next three lines before copying ./web in order to avoid running 'yarn install' every time some file in ./web changes
COPY ./web/package.json .
COPY ./web/yarn.lock .
RUN yarn install

COPY ./web .
RUN yarn build

FROM alpine
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY ./CHECKS .
COPY --from=gobuilder /src/github.com/florianloch/cassette/cassette .
COPY --from=web_distbuilder /build/dist ./web/dist
CMD ["./cassette"]
