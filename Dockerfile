FROM golang:alpine as builder
RUN apk update && apk add git && apk add binutils && apk add ca-certificates \
    && adduser -D -g '' user
COPY . /src
WORKDIR /src
RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo \
    -ldflags="-w -s" -o centerms . \
    && strip --strip-unneeded centerms

FROM alpine:latest
EXPOSE 8090 8080 8070
RUN apk --no-cache add curl
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /src/centerms /bin/centerms
USER user
ENTRYPOINT ["/bin/centerms"]
