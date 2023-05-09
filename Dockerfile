FROM alpine
COPY spacectl /usr/local/bin/spacectl
ENTRYPOINT ["/usr/local/bin/spacectl"]