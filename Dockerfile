FROM alpine

RUN set -xe \
    && apk add --no-cache git

COPY spacectl /usr/local/bin/spacectl
ENTRYPOINT ["/usr/local/bin/spacectl"]
