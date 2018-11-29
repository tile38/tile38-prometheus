FROM alpine:latest
RUN apk add --no-cache ca-certificates
ADD tile38-prometheus-sidekick /usr/local/bin
CMD tile38-prometheus-sidekick