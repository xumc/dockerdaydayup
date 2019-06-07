FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /app
ADD ./videoreport /app
EXPOSE 80
ENTRYPOINT ["/app/videoreport"]