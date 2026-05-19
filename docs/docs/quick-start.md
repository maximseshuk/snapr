# Quick Start

Install snapr, write a minimal config, and run your first backup.

## 1. Install

Two ways to get a `snapr` binary on your machine. For a containerised setup see [Docker](/deployment/docker).

### From GitHub Releases

Pre-built binaries for Linux and macOS are published on [GitHub Releases](https://github.com/maximseshuk/snapr/releases).

```bash
# Pick the right asset for your OS/architecture from the releases page,
# then extract it into a directory on your PATH.
curl -L -o snapr.tar.gz \
  https://github.com/maximseshuk/snapr/releases/latest/download/snapr_linux_amd64.tar.gz
tar -xzf snapr.tar.gz
sudo mv snapr /usr/local/bin/

snapr --help
```

Available archives:

- `snapr_linux_amd64.tar.gz`
- `snapr_linux_arm64.tar.gz`
- `snapr_darwin_amd64.tar.gz`
- `snapr_darwin_arm64.tar.gz`

### With `go install`

If you have Go 1.25+ installed:

```bash
go install github.com/maximseshuk/snapr/cmd/snapr@latest
```

The binary lands in `$(go env GOPATH)/bin`. Make sure that directory is on your `PATH`.

Pin a specific version:

```bash
go install github.com/maximseshuk/snapr/cmd/snapr@v1.0.0
```

### Build from source

For local development or customisation:

```bash
git clone https://github.com/maximseshuk/snapr.git
cd snapr
pnpm install
pnpm run build
./bin/snapr --help
```

Requires Go 1.25+ and [pnpm](https://pnpm.io) (used to build the embedded web UI).

## 2. Create a minimal config

Save the following as `snapr.yaml` in the current directory.

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

This backs up `/var/data` to `/var/backups` every day at 02:00 and keeps the last 7 archives.

## 3. Start snapr

```bash
SNAPR_ADMIN_PASSWORD=changeme snapr -config ./snapr.yaml
```

snapr will:

- start the HTTP server on `:8080`
- load the config and validate it
- register the `nightly-files` cron entry

## 4. Open the UI

Browse to <http://localhost:8080> and log in with `admin` / `changeme`.

From the dashboard you can:

- see job state and history
- trigger a job manually with **Run now**
- stream live logs
- download archived backups

## 5. Trigger a manual run

Click **Run now** on `nightly-files`. The first run also confirms that paths and permissions are correct without waiting until 02:00.

## Where to next

- [Docker](/deployment/docker) — run snapr in a container
- [Configuration](/configuration/) — full YAML reference
- [Jobs](/configuration/jobs) — schedule, retention, hook scripts
