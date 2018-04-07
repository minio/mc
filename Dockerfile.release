FROM alpine:3.7

MAINTAINER Minio Inc <dev@minio.io>

RUN \
    apk add --no-cache ca-certificates && \
    apk add --no-cache --virtual .build-deps curl && \
    curl https://dl.minio.io/client/mc/release/linux-amd64/mc > /usr/bin/mc && \
    chmod +x /usr/bin/mc && apk del .build-deps

ENTRYPOINT ["mc"]
