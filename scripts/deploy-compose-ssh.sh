#!/usr/bin/env bash
set -euo pipefail

required_vars=(
  DEPLOY_ENV
  DEPLOY_PATH
  SSH_HOST
  SSH_PORT
  SSH_USER
  SSH_PRIVATE_KEY
  FRONTEND_IMAGE
  BACKEND_IMAGE
  IMAGE_TAG
)

for var in "${required_vars[@]}"; do
  if [[ -z "${!var:-}" ]]; then
    echo "$var is required" >&2
    exit 2
  fi
done

shell_quote() {
  printf '%q' "$1"
}

key_file="${RUNNER_TEMP:-/tmp}/knowledgeos_deploy_key"
bundle_file="${RUNNER_TEMP:-/tmp}/knowledgeos_deploy_bundle.tgz"
remote_bundle="/tmp/knowledgeos_deploy_bundle_${GITHUB_RUN_ID:-manual}.tgz"
remote_path="$(shell_quote "$DEPLOY_PATH")"

mkdir -p "$HOME/.ssh"
printf '%s\n' "$SSH_PRIVATE_KEY" > "$key_file"
chmod 600 "$key_file"
ssh-keyscan -p "$SSH_PORT" -H "$SSH_HOST" >> "$HOME/.ssh/known_hosts"

tar -czf "$bundle_file" \
  docker-compose.deploy.yml \
  nginx/nginx.conf \
  scripts/deploy-compose.sh

ssh_base=(
  ssh
  -i "$key_file"
  -p "$SSH_PORT"
  -o IdentitiesOnly=yes
  -o StrictHostKeyChecking=yes
  "$SSH_USER@$SSH_HOST"
)

scp_base=(
  scp
  -i "$key_file"
  -P "$SSH_PORT"
  -o IdentitiesOnly=yes
  -o StrictHostKeyChecking=yes
)

"${ssh_base[@]}" "mkdir -p $remote_path"
if ! "${ssh_base[@]}" "test -f $remote_path/.env"; then
  echo "Missing $DEPLOY_PATH/.env on the server. Create it once before the first deploy." >&2
  exit 1
fi

"${scp_base[@]}" "$bundle_file" "$SSH_USER@$SSH_HOST:$remote_bundle"
"${ssh_base[@]}" "tar -xzf $(shell_quote "$remote_bundle") -C $remote_path && chmod +x $remote_path/scripts/deploy-compose.sh"

remote_command=$(cat <<EOF
cd $remote_path
DEPLOY_ENV=$(shell_quote "$DEPLOY_ENV") \\
COMPOSE_PROJECT_NAME=$(shell_quote "${COMPOSE_PROJECT_NAME:-knowledgeos-$DEPLOY_ENV}") \\
REGISTRY_USERNAME=$(shell_quote "${REGISTRY_USERNAME:-}") \\
REGISTRY_TOKEN=$(shell_quote "${REGISTRY_TOKEN:-}") \\
FRONTEND_IMAGE=$(shell_quote "$FRONTEND_IMAGE") \\
BACKEND_IMAGE=$(shell_quote "$BACKEND_IMAGE") \\
BACKUP_IMAGE=$(shell_quote "${BACKUP_IMAGE:-}") \\
IMAGE_TAG=$(shell_quote "$IMAGE_TAG") \\
SMOKE_URL=$(shell_quote "${SMOKE_URL:-}") \\
./scripts/deploy-compose.sh $(shell_quote "$DEPLOY_ENV")
EOF
)

"${ssh_base[@]}" "$remote_command"
