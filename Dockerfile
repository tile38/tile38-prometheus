FROM alpine:latest
RUN apk add --no-cache ca-certificates
ADD tile38-prometheus /usr/local/bin
CMD tile38-prometheus