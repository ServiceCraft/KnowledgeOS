#!/usr/bin/env sh
set -eu

DEPLOY_ENV="${1:-${DEPLOY_ENV:-}}"
if [ -z "$DEPLOY_ENV" ]; then
  echo "usage: $0 <prod|test>" >&2
  exit 2
fi

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.deploy.yml}"
ENV_FILE="${ENV_FILE:-.env}"
IMAGES_ENV_FILE="${IMAGES_ENV_FILE:-.env.images}"
COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-knowledgeos-$DEPLOY_ENV}"
REGISTRY="${REGISTRY:-ghcr.io}"

: "${FRONTEND_IMAGE:?FRONTEND_IMAGE is required}"
: "${BACKEND_IMAGE:?BACKEND_IMAGE is required}"

if [ -n "${REGISTRY_TOKEN:-}" ]; then
  : "${REGISTRY_USERNAME:?REGISTRY_USERNAME is required when REGISTRY_TOKEN is set}"
  printf '%s' "$REGISTRY_TOKEN" | docker login "$REGISTRY" -u "$REGISTRY_USERNAME" --password-stdin
fi

cat > "$IMAGES_ENV_FILE" <<EOF
IMAGE_TAG=${IMAGE_TAG:-}
FRONTEND_IMAGE=${FRONTEND_IMAGE}
BACKEND_IMAGE=${BACKEND_IMAGE}
BACKUP_IMAGE=${BACKUP_IMAGE:-}
EOF

compose() {
  docker compose \
    --env-file "$ENV_FILE" \
    --env-file "$IMAGES_ENV_FILE" \
    -f "$COMPOSE_FILE" \
    -p "$COMPOSE_PROJECT_NAME" \
    "$@"
}

compose pull
compose up -d --remove-orphans
compose ps

compose exec -T app /bin/sh -c 'wget -qO- http://127.0.0.1:8080/healthz >/dev/null'

if [ -n "${SMOKE_URL:-}" ]; then
  if command -v curl >/dev/null 2>&1; then
    curl -fsS --retry 5 --retry-delay 2 "$SMOKE_URL/" >/dev/null
  else
    wget -qO- "$SMOKE_URL/" >/dev/null
  fi
fi
