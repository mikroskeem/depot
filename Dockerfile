FROM golang:1.12.6-alpine3.10

# Install required dependencies
RUN    apk update \
    && apk add git binutils

# Build depot
USER nobody
WORKDIR /tmp
RUN git clone --depth=1 https://github.com/mikroskeem/depot.git depot \
    && cd depot \
    && env GOCACHE=/tmp/.cache CGO_ENABLED=0 GOOS=linux go build \
    && strip --strip-unneeded depot

# Create depot image
FROM scratch
COPY --from=0 /tmp/depot/depot /depot

USER 99:99
VOLUME /data
EXPOSE 8080/tcp
CMD ["/depot", "-config", "/data/config.toml"]
