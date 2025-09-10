# Pull thor into a second stage deploy alpine container
FROM alpine:3.21.3

RUN apk add --no-cache ca-certificates
RUN apk upgrade libssl3 libcrypto3
COPY $TARGETPLATFORM/thor /usr/local/bin/thor
COPY $TARGETPLATFORM/disco /usr/local/bin/disco
RUN adduser -D -s /bin/ash thor
USER thor

EXPOSE 8669 11235 11235/udp 55555/udp
ENTRYPOINT ["thor"]
