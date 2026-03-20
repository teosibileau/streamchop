FROM alpine:3.21

RUN apk add --no-cache ffmpeg

RUN mkdir /output

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
