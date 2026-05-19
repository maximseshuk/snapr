# Dev Environment

Local stack for testing snapr: Postgres, MinIO (S3), SFTP, WebDAV, sample data.

## Files

| File                 | Purpose                                             |
| -------------------- | --------------------------------------------------- |
| `docker-compose.yml` | Postgres + MinIO + SFTP + WebDAV + auto-bucket init |
| `snapr.yaml`         | Dev config with sample jobs                         |
| `init-db.sql`        | Postgres schema + seed data                         |
| `data/`              | Sample files for local-files jobs                   |
| `backups/`           | Output dir for `local` storage                      |

## Endpoints

| Service       | URL                    | Credentials                            |
| ------------- | ---------------------- | -------------------------------------- |
| API           | http://localhost:47100 | —                                      |
| Web UI        | http://localhost:47101 | —                                      |
| Postgres      | localhost:47102        | `postgres` / `postgres` (db: `testdb`) |
| MinIO API     | http://localhost:47103 | `minioadmin` / `minioadmin`            |
| MinIO Console | http://localhost:47104 | `minioadmin` / `minioadmin`            |
| SFTP          | localhost:47105        | `backup` / `backup`                    |
| WebDAV        | http://localhost:47106 | `backup` / `backup`                    |

Bucket `backups` is created on first start.

## Quick start

```bash
pnpm run stack:up   # postgres + minio + sftp + webdav
pnpm run dev        # api + web
```

Open http://localhost:47101 → hit **Run** on a job → check MinIO Console at http://localhost:47104.

## Jobs

- **`files-to-s3`** — `data/` → MinIO `backups/local-backup/`. Retention 7.
- **`database-snapshot`** — `testdb` → `backups/` + MinIO `backups/postgres/`. AES-256-CBC encryption, retention 30.
- **`files-to-sftp`** — `data/` → SFTP `/snapr/`. Retention 14.
- **`files-to-webdav`** — `data/` → WebDAV `/snapr/`. AES-256-CBC encryption, retention 7.
- **`large-archive-split`** — generates 50 MB payload, splits into 10 MB parts. Retention 3.

## Common operations

```bash
# Watch logs
pnpm run stack:logs

# Wipe volumes
pnpm run stack:reset

# Inspect Postgres
docker exec -it snapr-dev-postgres psql -U postgres -d testdb

# Trigger via API
curl -X POST http://localhost:47100/api/v1/jobs/files-to-s3/run
curl http://localhost:47100/api/v1/jobs/files-to-s3/status
```

## Adding test data

```bash
echo "new content" > data/new.txt
echo "ignored" > data/skip.log   # matches excludes
```
