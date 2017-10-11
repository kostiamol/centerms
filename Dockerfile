FROM alpine
MAINTAINER Molchanov Kostiantyn (kostyamol@gmail.com)

EXPOSE 6379 3030 3000 8100 2540

RUN \
    mkdir -p /home/centerms/bin \
    mkdir -p /home/centerms/view

WORKDIR /home/centerms/bin
COPY ./cmd/centerms .
COPY ./view ../view

RUN \  
    chown daemon center-smart-house && \
    chmod +x center-smart-house
  
USER daemon
ENTRYPOINT ["./centerms"]
