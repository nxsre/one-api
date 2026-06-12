# syntax=docker/dockerfile:1.6
FROM golang:1.26-trixie AS build
WORKDIR /src
COPY go.mod main.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /mock-upstream .

FROM alpine:3.24
RUN apk add --no-cache wget
COPY --from=build /mock-upstream /usr/local/bin/mock-upstream
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/mock-upstream"]
