FROM golang:1.26.0-alpine AS build
WORKDIR /go/src/github.com/the-pilot-club/online-checker
COPY ./ ./
RUN go build -o bin/presence-checker ./cmd/presence-checker

FROM alpine:latest AS app
RUN apk add --no-cache ca-certificates \
    && adduser -D -H -u 10001 app
WORKDIR /app
COPY --from=build /go/src/github.com/the-pilot-club/online-checker/bin/presence-checker ./
USER app
EXPOSE 8080
ENTRYPOINT ["./presence-checker"]
