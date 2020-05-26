FROM golang:1.14.3-alpine3.11

# Build depot
WORKDIR $GOPATH/src/github.com/mikroskeem/depot
COPY . ./
RUN env CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -ldflags="-s -w" -o /depot

# Create depot image
FROM scratch
COPY --from=0 /depot /depot

USER 99:99
VOLUME /data
EXPOSE 8080/tcp
CMD ["/depot", "-config", "/data/config.toml"]
