FROM golang AS gobuilder
WORKDIR /src/github.com/florianloch/cassette
COPY . .
RUN GOOS=linux GARCH=amd64 CGO_ENABLED=0 go build -o cassette .


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