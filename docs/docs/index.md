---
pageType: doc
title: Introduction
---

# snapr

Self-hosted backup service. Schedules backups of files and databases, compresses, encrypts, and uploads to one or more storage backends. Comes with a web UI and a REST API.

## Highlights

- Cron-scheduled jobs with manual trigger
- Multiple sources per job — local files, Postgres / MySQL / MariaDB, MongoDB, Redis, SQLite, S3, Bunny Storage
- Multiple storages per job — local, S3-compatible, SFTP, WebDAV, Bunny
- Compression — `tar`, `tar.gz`, `gzip`, `zip`
- OpenSSL symmetric encryption
- Webhook / Telegram / Email notifiers
- Env-var references for secrets

## Where to next

- [Quick Start](/quick-start) — install and run your first backup
- [Docker](/deployment/docker) — run snapr in a container
- [Configuration](/configuration/) — full YAML reference
- [Jobs](/configuration/jobs) — schedule, retention, hook scripts
- [API reference](/api/) — JSON HTTP API for scripting and integrations
