FROM golang:1.7-alpine

WORKDIR /go/src/app

COPY . /go/src/app

RUN \
	apk add --no-cache git && \
	go-wrapper download && \
	go-wrapper install -ldflags "-X github.com/minio/mc/cmd.Version=2017-02-02T22:38:48Z -X github.com/minio/mc/cmd.ReleaseTag=RELEASE.2017-02-02T22-38-48Z -X github.com/minio/mc/cmd.CommitID=ad850412d25d2b26b83949a4794cf33b916b97e1" && \
	rm -rf /go/pkg /go/src && \
	apk del git

ENTRYPOINT ["mc"]
