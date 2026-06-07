FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api ./cmd/api

FROM alpine:3.19

# postgresql16-client provides pg_dump for the backup snapshot endpoint
# (must match the PostgreSQL 16 server). ca-certificates/tzdata for runtime.
RUN apk add --no-cache ca-certificates tzdata postgresql16-client

COPY --from=builder /bin/api /bin/api
COPY migrations /migrations

# Bake source so the snapshot endpoint can serve code.tar.gz even without a
# mount. NOTE: the build context is ./backend, so this baked /app/src contains
# the BACKEND source only — a complete full-project snapshot requires mounting
# the repository at /src (docker-compose sets BACKUP_CODE_PATH=/src). Secrets
# and binaries are filtered at snapshot time regardless of source path.
COPY --from=builder /app /app/src

# Record the build commit for metadata.json.
ARG GIT_COMMIT=unknown
RUN mkdir -p /app && printf '%s\n' "$GIT_COMMIT" > /app/COMMIT

ENV BACKUP_CODE_PATH=/app/src \
    BACKUP_COMMIT_FILE=/app/COMMIT

EXPOSE 8080

CMD ["/bin/api"]
