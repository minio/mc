FROM golang:1.11.4-alpine3.7

LABEL maintainer="Minio Inc <dev@minio.io>"

ENV GOPATH /go
ENV CGO_ENABLED 0

WORKDIR /go/src/github.com/minio/

RUN  \
     apk add --no-cache git && \
     go get -v -d github.com/minio/mc && \
     cd /go/src/github.com/minio/mc && \
     go install -v -ldflags "$(go run buildscripts/gen-ldflags.go)"

FROM alpine:3.7

COPY --from=0 /go/bin/mc /usr/bin/mc

RUN  \
     apk add --no-cache ca-certificates && \
     echo 'hosts: files mdns4_minimal [NOTFOUND=return] dns mdns4' >> /etc/nsswitch.conf

ENTRYPOINT ["mc"]
