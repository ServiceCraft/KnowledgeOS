# Build a static relay binary and run it on a minimal image.
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY main.go ./
RUN CGO_ENABLED=0 go build -o /relay .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /relay /relay
EXPOSE 8080
ENTRYPOINT ["/relay"]
