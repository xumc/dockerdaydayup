FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /app
ADD ./videoapi /app
EXPOSE 80
ENTRYPOINT ["/app/videoapi"]