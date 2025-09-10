# Pull thor into a second stage deploy alpine container
FROM alpine:3.21.3

ARG TARGETPLATFORM
ARG BINARY

RUN echo "Building $TARGETPLATFORM/$BINARY"

RUN apk add --no-cache ca-certificates
RUN apk upgrade libssl3 libcrypto3
COPY $TARGETPLATFORM/$BINARY /usr/local/bin/$BINARY
RUN adduser -D -s /bin/ash thor
USER thor

EXPOSE 8669 11235 11235/udp 55555/udp
ENTRYPOINT ["thor"]
