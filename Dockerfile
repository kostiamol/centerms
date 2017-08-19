FROM alpine
MAINTAINER giperboloid (kostyamol@gmail.com)

EXPOSE 6379 3030 3000 8100 2540

RUN \
    mkdir -p /home/center-smart-house/bin \
    mkdir -p /home/center-smart-house/view

WORKDIR /home/center-smart-house/bin
COPY ./cmd/center-smart-house .
COPY ./view ../view

RUN \  
    chown daemon center-smart-house && \
    chmod +x center-smart-house
  
USER daemon
ENTRYPOINT ["./center-smart-house"]
