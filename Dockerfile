FROM golang:1.17-alpine as builder

LABEL maintainer="MinIO Inc <dev@min.io>"

ENV GOPATH /go
ENV CGO_ENABLED 0
ENV GO111MODULE on

RUN  \
     apk add --no-cache git && \
     git clone https://github.com/minio/mc && cd mc && \
     go install -v -ldflags "$(go run buildscripts/gen-ldflags.go)"

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.5

ARG TARGETARCH

COPY --from=builder /go/bin/mc /usr/bin/mc
COPY --from=builder /go/mc/CREDITS /licenses/CREDITS
COPY --from=builder /go/mc/LICENSE /licenses/LICENSE

RUN  \
     microdnf update --nodocs && \
     microdnf install ca-certificates --nodocs && \
     microdnf clean all

ENTRYPOINT ["mc"]
