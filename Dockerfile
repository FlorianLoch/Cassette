FROM golang AS gobuilder
WORKDIR /src/github.com/florianloch/spotistate
COPY . .
RUN GOOS=linux GARCH=amd64 CGO_ENABLED=0 go build -o spotistate .

FROM node AS web_distbuilder
WORKDIR /build
COPY ./web .
# Don't use dependencies that have been built on the host system...
RUN rm -rf node_modules
RUN npm install
RUN npm run build

FROM alpine
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=gobuilder /src/github.com/florianloch/spotistate/spotistate .
COPY --from=web_distbuilder /build/dist ./web/dist
CMD ["./spotistate"]