FROM golang:1.8-alpine

WORKDIR /go/src/app

COPY . /go/src/app

RUN \
	apk add --no-cache git && \
	go-wrapper download && \
	go-wrapper install -ldflags "-X github.com/minio/mc/cmd.Version=2017-04-03T18:35:01Z -X github.com/minio/mc/cmd.ReleaseTag=RELEASE.2017-04-03T18-35-01Z -X github.com/minio/mc/cmd.CommitID=81d17d28ec28987ba3c57349fc57b4c064bb5b37" && \
	rm -rf /go/pkg /go/src && \
	apk del git

ENTRYPOINT ["mc"]
