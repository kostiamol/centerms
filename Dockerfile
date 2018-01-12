FROM alpine:latest
MAINTAINER Kostiantyn Molchanov (kostyamol@gmail.com)

RUN apk --no-cache add ca-certificates
EXPOSE 3092 3126 3000 3301 3546

WORKDIR /root/
COPY ./cmd/centerms/centerms .

RUN \  
    chown daemon centerms && \
    chmod +x centerms  
USER daemon

ENTRYPOINT ["./centerms"]
