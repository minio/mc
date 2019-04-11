FROM alpine:3.9

MAINTAINER MinIO Inc <dev@min.io>

RUN \
    apk add --no-cache ca-certificates && \
    apk add --no-cache --virtual .build-deps curl && \
    curl https://dl.min.io/client/mc/release/linux-amd64/mc > /usr/bin/mc && \
    chmod +x /usr/bin/mc && apk del .build-deps

ENTRYPOINT ["mc"]
