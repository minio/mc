FROM golang:1.9-alpine

WORKDIR /go/src/app

COPY . /go/src/app

RUN \
	apk add --no-cache git && \
	go-wrapper download && \
	go-wrapper install -ldflags "$(go run buildscripts/gen-ldflags.go)" && \
	rm -rf /go/pkg /go/src && \
	apk del git

ENTRYPOINT ["mc"]
