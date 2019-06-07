FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /app
ADD ./videoapi /app
EXPOSE 8080
ENTRYPOINT ["/app/videoapi"]