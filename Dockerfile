FROM golang:1.13-alpine as build

LABEL maintainer="MinIO Inc <dev@min.io>"

ENV GOPATH /go
ENV CGO_ENABLED 0
ENV GO111MODULE on

RUN  \
     apk add --no-cache git && \
     git clone https://github.com/minio/mc && cd mc && \
     go install -v -ldflags "$(go run buildscripts/gen-ldflags.go)"

FROM alpine:3.12

COPY --from=build /go/bin/mc /usr/bin/mc

RUN  \
     apk add --no-cache ca-certificates 'curl>7.61.0' && \
     curl -s -q -O https://raw.githubusercontent.com/minio/minio/master/CREDITS

ENTRYPOINT ["mc"]
