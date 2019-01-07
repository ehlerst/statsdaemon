FROM golang:1.11-alpine AS build-env

RUN apk add git dep gcc libc-dev --no-cache

RUN mkdir -p /go/src/github.com/raintank/statsdaemon

COPY . /go/src/github.com/raintank/statsdaemon

WORKDIR /go/src/github.com/raintank/statsdaemon

RUN mkdir -p build

RUN cd cmd/statsdaemon; GOOS=linux go build -o /go/src/github.com/raintank/statsdaemon/build/statsdaemon

FROM alpine
MAINTAINER Tim Ehlers <ehlerst@yahoo.com>

RUN apk --update add ca-certificates

COPY --from=build-env  /go/src/github.com/raintank/statsdaemon/statsdaemon.ini /etc/statsdaemon.ini
COPY --from=build-env  /go/src/github.com/raintank/statsdaemon/build/statsdaemon /usr/bin/statsdaemon
COPY --from=build-env  /go/src/github.com/raintank/statsdaemon/scripts/entrypoint.sh /usr/bin/

EXPOSE 8125
EXPOSE 8126

ENTRYPOINT ["/usr/bin/entrypoint.sh"]