FROM golang:1.26.0-alpine AS build
WORKDIR /go/src/github.com/the-pilot-club/online-checker
COPY ./ ./
RUN go build -o bin/bot main.go
RUN go build -o bin/bot-atc atc-main.go

FROM alpine:latest AS app
WORKDIR /app
COPY --from=build /go/src/github.com/the-pilot-club/online-checker/bin/* ./