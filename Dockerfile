FROM golang:1.24.1-alpine AS build
WORKDIR /go/src/github.com/the-pilot-club/online-checker
COPY ./ ./
RUN go build -o bin/bot .
ENTRYPOINT ["bin/bot"]