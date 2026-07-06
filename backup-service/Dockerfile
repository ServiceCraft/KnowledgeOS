FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod ./
# No external dependencies — go.sum is not required.
COPY *.go ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /bin/backup-service .

FROM alpine:3.19

# git is required to push code to GitHub; ca-certificates for HTTPS.
RUN apk add --no-cache git ca-certificates tzdata

COPY --from=builder /bin/backup-service /bin/backup-service

# Default data volume for versioned data snapshots.
VOLUME ["/data"]

# Cron mode by default; override the command with --once for a single run.
ENTRYPOINT ["/bin/backup-service"]
