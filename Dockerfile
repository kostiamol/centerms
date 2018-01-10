FROM alpine:latest
MAINTAINER Kostiantyn Molchanov (kostyamol@gmail.com)

RUN apk --no-cache add ca-certificates

EXPOSE 6379 3030 3000 8100 2540

RUN \
    mkdir -p /home/centerms/bin \
    mkdir -p /home/centerms/view

WORKDIR /home/centerms/bin
COPY ./cmd/centerms .

RUN \  
    chown daemon centerms && \
    chmod +x centerms
  
USER daemon
ENTRYPOINT ["./centerms"]
