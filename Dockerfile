FROM alpine

ARG TARGETPLATFORM

RUN set -xe \
    && apk add --no-cache git

COPY $TARGETPLATFORM/spacectl /usr/local/bin/spacectl
ENTRYPOINT ["/usr/local/bin/spacectl"]
