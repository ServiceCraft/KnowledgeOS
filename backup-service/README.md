# KnowledgeOS Backup Service

Standalone service that backs up the main KnowledgeOS application:

- pulls a snapshot from `GET /api/v1/backup/snapshot`,
- pushes the application code to a GitHub repository (branch + timestamp tag),
- stores versioned, gzipped DB dumps locally with FIFO rotation.

No external Go dependencies (stdlib only). Requires the `git` CLI at runtime
(provided by the Docker image).

## Usage

```bash
# single cycle (debug / on-demand)
./backup-service --once

# scheduled mode (cron from SCHEDULE_CRON, UTC)
./backup-service
```

Configuration and the full backup/restore guide:
see [`../docs/BACKUP.md`](../docs/BACKUP.md).
