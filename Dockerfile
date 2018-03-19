FROM golang:alpine as builder
RUN apk update && apk add git && apk add binutils && apk add ca-certificates
RUN adduser -D -g '' appuser
COPY . $GOPATH/src/github.com/kostiamol/centerms
WORKDIR $GOPATH/src/github.com/kostiamol/centerms
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o $GOPATH/bin/centerms ./cmd/centerms
RUN cd $GOPATH/bin \
    strip --strip-unneeded centerms

FROM scratch
EXPOSE 3092 3126 3301 3546
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /go/bin/centerms /go/bin/centerms
USER appuser
ENTRYPOINT ["/go/bin/centerms"]
