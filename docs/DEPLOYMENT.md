# CI/CD Deployment

KnowledgeOS is deployed by GitHub Actions with Docker images and Docker Compose.
The `main` branch deploys to production. Every other pushed branch deploys to the
shared test stand and replaces the previous test deployment.

## Pipeline

1. `ci` runs for pull requests and pushes:
   - `cd backend && go test ./...`
   - `cd backup-service && go test ./...`
   - `golangci-lint run ./...` in `backend`
   - `cd frontend && npm ci && npm run lint && npm run build`
2. `build` runs only for pushes and publishes images to GHCR:
   - `ghcr.io/<owner>/<repo>/backend:sha-<commit>`
   - `ghcr.io/<owner>/<repo>/frontend:sha-<commit>`
   - `ghcr.io/<owner>/<repo>/backup-service:sha-<commit>`
3. `deploy_test` runs for non-`main` branches with GitHub Environment `test`.
4. `deploy_prod` runs for `main` with GitHub Environment `prod`.

Production should use an environment protection rule with manual approval before
the `deploy_prod` job is allowed to access production secrets.

## Server Layout

Prepare one directory per environment:

```bash
sudo mkdir -p /opt/knowledgeos/prod
sudo mkdir -p /opt/knowledgeos/test
sudo chown -R deploy:deploy /opt/knowledgeos
```

GitHub Actions uploads these files to the target directory before every deploy:

- `docker-compose.deploy.yml`
- `nginx/nginx.conf`
- `scripts/deploy-compose.sh`
- `.env` generated from the GitHub Environment secret `APP_ENV_FILE`
- `.env.images` generated on the server with the image tags for the current run

The server must have Docker Engine and the Docker Compose plugin installed.

## GitHub Environments

Create two GitHub Environments: `test` and `prod`.

Add these secrets to each environment:

- `SSH_HOST`: server hostname or IP.
- `SSH_PORT`: SSH port, usually `22`.
- `SSH_USER`: deploy user.
- `SSH_PRIVATE_KEY`: private key for the deploy user.
- `DEPLOY_PATH`: target directory, for example `/opt/knowledgeos/prod`.
- `APP_ENV_FILE`: full runtime `.env` content for this environment.
- `REGISTRY_USERNAME`: registry user. For GHCR this can be a GitHub username.
- `REGISTRY_TOKEN`: token with package read permission from the server.

Add these optional environment variables:

- `COMPOSE_PROJECT_NAME`: `knowledgeos-prod` or `knowledgeos-test`.
- `SMOKE_URL`: public URL used by the post-deploy smoke check.

## Runtime Environment

`APP_ENV_FILE` should be based on `.env.example` and must be different for
production and test.

Required application values:

- `APP_PROFILE`
- `POSTGRES_HOST`
- `POSTGRES_PORT`
- `POSTGRES_USER`
- `POSTGRES_PASSWORD`
- `POSTGRES_DB`
- `JWT_SECRET`
- `SUPERADMIN_EMAIL`
- `SUPERADMIN_PASSWORD`
- `SECRETS_ENCRYPTION_KEY`

AI and bot values, if those features are enabled:

- `YANDEX_ENDPOINT`
- `YANDEX_FOLDER_ID`
- `YANDEX_API_KEY`
- `YANDEX_DEFAULT_CHAT_MODEL_LITE`
- `YANDEX_DEFAULT_CHAT_MODEL_PRO`
- `YANDEX_EMBEDDING_DOC_MODEL`
- `YANDEX_EMBEDDING_QUERY_MODEL`
- `YANDEX_TIMEOUT_SECONDS`
- `YANDEX_MAX_RETRIES`

Backup values, if the backup endpoint or backup service is enabled:

- `BACKUP_API_KEY`
- `BACKUP_APP_URL`
- `GITHUB_REPO_URL`
- `GITHUB_TOKEN`
- `GITHUB_BRANCH`
- `BACKUPS_KEEP`
- `SCHEDULE_CRON`
- `HTTP_TIMEOUT_SECONDS`

Sync-agent values are not included in the default deploy compose. If the sync
agent is deployed later, prepare `CLOUD_API_URL`, `CLOUD_API_KEY`, and
`SYNC_INTERVAL_SECONDS`.

## Domains and TLS

Point the production and test domains at the corresponding server or reverse
proxy. The deploy compose exposes frontend nginx through `FRONTEND_PORT`.

Recommended defaults:

- Production: terminate TLS at an external reverse proxy on `443`, forward to
  `127.0.0.1:8080`.
- Test: terminate TLS the same way, forward to `127.0.0.1:8081` if both
  environments share a server.

The repository nginx config is HTTP-only by default. If TLS is terminated inside
the container, mount certificates and update `nginx/nginx.conf` for that server.

## Manual Deploy

After the target directory contains `docker-compose.deploy.yml`, `nginx/nginx.conf`,
`scripts/deploy-compose.sh`, `.env`, and `.env.images`, a deploy can be run by
hand:

```bash
cd /opt/knowledgeos/prod
COMPOSE_PROJECT_NAME=knowledgeos-prod ./scripts/deploy-compose.sh prod
```

For test:

```bash
cd /opt/knowledgeos/test
COMPOSE_PROJECT_NAME=knowledgeos-test ./scripts/deploy-compose.sh test
```

## Rollback

Rollback is a redeploy with older image tags:

1. Find the previous successful `sha-<commit>` image tag in GHCR.
2. Update `.env.images` on the server with the previous backend, frontend, and
   backup-service image tags.
3. Run `./scripts/deploy-compose.sh prod` or `./scripts/deploy-compose.sh test`.

Database migrations currently run during backend startup. Before production
rollback, check whether the deployed commit introduced irreversible migrations.

## Operational Notes

- Non-`main` branches share one test stand, so the latest successful branch push
  replaces earlier test deployments.
- Keep production and test isolated with separate compose project names, volumes,
  env files, domains, and ideally separate servers.
- Do not store `.env`, private keys, registry tokens, or Yandex credentials in the
  repository.
- Monitor disk usage on Docker hosts because images, volumes, and database files
  can grow quickly.
