FROM golang:1.7-alpine

WORKDIR /go/src/app

COPY . /go/src/app

RUN \
	apk add --no-cache git && \
	go-wrapper download && \
	go-wrapper install -ldflags "-X github.com/minio/mc/cmd.Version=2017-02-06T20:16:19Z -X github.com/minio/mc/cmd.ReleaseTag=RELEASE.2017-02-06T20-16-19Z -X github.com/minio/mc/cmd.CommitID=2c8115de4edc5612525488b1b3b804689d336d01" && \
	rm -rf /go/pkg /go/src && \
	apk del git

ENTRYPOINT ["mc"]
