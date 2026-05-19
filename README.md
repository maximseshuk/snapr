<div align="center">

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/docs/public/logo-light.svg">
  <img src="docs/docs/public/logo-dark.svg" alt="snapr" height="80" />
</picture>

<h1>snapr</h1>

<p>Self-hosted backup service for files and databases. Runs on a schedule, compresses and (optionally) encrypts your data, and uploads it to one or more storage backends. Comes with a web UI and a REST API.</p>

<a href="https://github.com/maximseshuk/snapr/releases/"><img src="https://img.shields.io/github/v/release/maximseshuk/snapr?style=flat-square&logo=github" alt="GitHub release" /></a>
<a href="https://github.com/maximseshuk/snapr/pkgs/container/snapr"><img src="https://img.shields.io/badge/ghcr.io-snapr-2496ed?style=flat-square&logo=docker&logoColor=white" alt="Docker image" /></a>
<a href="https://github.com/maximseshuk/snapr/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/maximseshuk/snapr/ci.yml?style=flat-square&logo=github" alt="CI" /></a>
<a href="https://snapr.seshuk.im/"><img src="https://img.shields.io/badge/docs-snapr.seshuk.im-blue?style=flat-square&logo=readthedocs&logoColor=white" alt="Documentation" /></a>
<a href="https://github.com/maximseshuk/snapr/blob/main/LICENSE"><img src="https://img.shields.io/github/license/maximseshuk/snapr?style=flat-square" alt="license" /></a>
<a href="https://ko-fi.com/V7V61UCT39"><img src="https://img.shields.io/badge/Ko--fi-Buy_me_a_coffee-ff5f5f?style=flat-square&logo=ko-fi&logoColor=white" alt="Ko-fi" /></a>

</div>

snapr runs backup jobs on a schedule. Each job reads from one or more **sources** (local files, Postgres, MySQL/MariaDB, MongoDB, Redis, SQLite, S3, Bunny), packs the result into an archive, and uploads it to one or more **storages** (local disk, S3-compatible, SFTP, WebDAV, Bunny). The web UI and REST API let you see job status, follow live logs, run a job manually, and download archives.

## Features

- One job can read from many sources and upload to many storages.
- Database dumps for Postgres, MySQL, MariaDB, MongoDB, Redis, and SQLite. Plus plain local files.
- Compression: `tar`, `tar.gz`, `gzip`, `zip`. You can split large archives into parts.
- Optional encryption with OpenSSL (AES-256-CBC and similar ciphers).
- Retention per storage — keep the last N runs.
- Notifications via webhook, Telegram, or email — on success, failure, or both.
- Web UI and REST API. OpenAPI spec at `/api/v1/openapi`.
- Run a shell script before or after a job (`beforeScript` / `afterScript`).
- One Go binary, no runtime. You only need the dump tools for the databases you back up (`pg_dump`, `mysqldump`, `mongodump`, `redis-cli`).

## Quick start

### Docker

```bash
docker run -d \
  --name snapr \
  -p 8080:8080 \
  -e SNAPR_ADMIN_PASSWORD=changeme \
  -v $(pwd)/snapr.yaml:/etc/snapr/snapr.yaml:ro \
  -v $(pwd)/backups:/var/backups \
  ghcr.io/maximseshuk/snapr:latest
```

Open <http://localhost:8080> and sign in as `admin` with the password you set.

### Binary

Download a pre-built archive from [Releases](https://github.com/maximseshuk/snapr/releases) (Linux and macOS, amd64 and arm64), or install with Go:

```bash
go install github.com/maximseshuk/snapr/cmd/snapr@latest
snapr -config ./snapr.yaml
```

### Minimal config

```yaml
server:
  address: '0.0.0.0:8080'
  auth:
    enabled: true
    username: admin
    password: env:SNAPR_ADMIN_PASSWORD

jobs:
  - name: nightly-files
    schedule: '0 2 * * *'
    compression: tar.gz
    sources:
      - type: local
        path: /var/data
    storages:
      - name: local-disk
        type: local
        path: /var/backups
    retention:
      last: 7
```

This config backs up `/var/data` to `/var/backups` every night at 02:00 and keeps the last 7 archives. For everything else see the [Quick Start](https://snapr.seshuk.im/quick-start) and the [full configuration reference](https://snapr.seshuk.im/configuration/).

## Local development

You need Go 1.25+, Node 22+, and pnpm 10+.

```bash
git clone https://github.com/maximseshuk/snapr.git
cd snapr
pnpm install

pnpm run stack:up   # starts MinIO, Postgres, SFTP, and WebDAV in Docker
pnpm run dev        # snapr API on :47100, web UI on :5173
```

The dev stack uses the sample config [`examples/dev/snapr.yaml`](examples/dev/snapr.yaml), which has a job for every storage backend.

> [!NOTE]
> `pnpm run dev` runs the Go API and the Vite dev server together. The Docker stack only starts the storage targets — snapr itself runs from source on your machine.

Useful scripts:

| Command                                            | What it does                                          |
| -------------------------------------------------- | ----------------------------------------------------- |
| `pnpm run build`                                   | Build the web UI and the Go binary into `./bin/snapr` |
| `pnpm run stack:up` / `stack:down` / `stack:reset` | Manage the Docker dev stack                           |
| `pnpm run test` / `test:race` / `test:cover`       | Run the Go test suite                                 |
| `pnpm run lint` / `lint:fix`                       | Lint Go and web code                                  |
| `pnpm run format` / `format:check`                 | Format Go and web code                                |
| `pnpm run docs:dev` / `docs:build`                 | Run or build the docs site                            |

## Documentation

Full docs are at **<https://snapr.seshuk.im/>**:

- [Quick start](https://snapr.seshuk.im/quick-start)
- [Configuration reference](https://snapr.seshuk.im/configuration/)
- [Docker deployment](https://snapr.seshuk.im/deployment/docker)
- [API reference](https://snapr.seshuk.im/api/)

Source files for the docs live in [`docs/docs/`](docs/docs/).

## Support

Bug reports, feature requests, and questions go to [GitHub Issues](https://github.com/maximseshuk/snapr/issues).

## License

MIT — see [LICENSE](LICENSE).

## Credits

Built by [Maxim Seshuk](https://github.com/maximseshuk).

If snapr saves you time, you can [buy me a coffee](https://ko-fi.com/V7V61UCT39) ☕
