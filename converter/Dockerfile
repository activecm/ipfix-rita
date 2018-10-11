FROM golang:1.10-alpine as ipfix-rita-builder
RUN apk add --no-cache git make ca-certificates wget build-base
RUN wget -q -O /go/bin/dep https://github.com/golang/dep/releases/download/v0.5.0/dep-linux-amd64 && chmod +x /go/bin/dep
WORKDIR /go/src/github.com/activecm/ipfix-rita/converter
COPY . .
RUN make CGO_ENABLED=0 GOARCH=amd64 GOOS=linux
RUN make install

FROM alpine:3.8

RUN apk add --no-cache tzdata

# Use WORKDIR to create /etc/ipfix-rita since "mkdir" doesn't exist in scratch
WORKDIR /etc/ipfix-rita

# Use a bind mount of docker config in swarm mode instead
# of copying the default configuration into the image.
# COPY --from=ipfix-rita-builder /etc/ipfix-rita/* .

WORKDIR /

COPY --from=ipfix-rita-builder /usr/local/bin/ipfix-rita-converter ipfix-rita-converter
ENTRYPOINT ["/ipfix-rita-converter"]
