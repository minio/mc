FROM golang:1.12-alpine

LABEL maintainer="Minio Inc <dev@minio.io>"

ENV GOPATH /go
ENV CGO_ENABLED 0
ENV GO111MODULE on

RUN  \
     apk add --no-cache git && \
     git clone https://github.com/minio/mc && cd mc && \
     go install -v -ldflags "$(go run buildscripts/gen-ldflags.go)"

FROM alpine:3.9

COPY --from=0 /go/bin/mc /usr/bin/mc

RUN  \
     apk add --no-cache ca-certificates && \
     echo 'hosts: files mdns4_minimal [NOTFOUND=return] dns mdns4' >> /etc/nsswitch.conf

ENTRYPOINT ["mc"]
