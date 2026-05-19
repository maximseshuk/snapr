# Docker

Run snapr in a container instead of installing the binary on the host.

## Docker

```bash
docker run -d \
  --name snapr \
  -p 8080:8080 \
  -e SNAPR_ADMIN_PASSWORD=changeme \
  -v $(pwd)/snapr.yaml:/etc/snapr/snapr.yaml:ro \
  -v $(pwd)/backups:/var/backups \
  ghcr.io/maximseshuk/snapr:latest
```

snapr looks for `snapr.yaml` in `/etc/snapr/` by default inside the container.

## Docker Compose

The repo ships a dev stack with MinIO (S3), Postgres, SFTP, and WebDAV containers — useful for trying every storage backend locally. The stack only starts the storage targets; snapr itself runs from source via `pnpm run dev` (see [examples/dev/README.md](https://github.com/maximseshuk/snapr/blob/main/examples/dev/README.md)).

```bash
git clone https://github.com/maximseshuk/snapr.git
cd snapr
pnpm install
pnpm run stack:up    # start storage containers
pnpm run dev         # run snapr against ./examples/dev/snapr.yaml
```

The dev config binds the API on `:47100` with `admin` / `admin` — for local testing only; never reuse these in production.

A minimal compose for production looks like this:

```yaml
services:
  snapr:
    image: ghcr.io/maximseshuk/snapr:latest
    restart: unless-stopped
    ports:
      - '8080:8080'
    environment:
      SNAPR_ADMIN_PASSWORD: ${SNAPR_ADMIN_PASSWORD}
      SNAPR_JWT_SECRET: ${SNAPR_JWT_SECRET}
    volumes:
      - ./snapr.yaml:/etc/snapr/snapr.yaml:ro
      - ./backups:/var/backups
```

## Volumes & paths

Inside the container snapr runs as user `snapr` (uid `1001`). Mount:

- the config file read-only at `/etc/snapr/snapr.yaml`
- any **source** path you want to back up — read-only is fine
- any **local storage** path — must be writable by uid `1001`. If the host directory is owned by another user, either `chown 1001` it or run the container with a matching `--user`.

## Environment variables

Use them with `env:` references in `snapr.yaml`:

```yaml
server:
  secret: env:SNAPR_JWT_SECRET
  auth:
    password: env:SNAPR_ADMIN_PASSWORD

jobs:
  - sources:
      - type: postgresql
        password: env:PG_PASSWORD
```

See [Configuration → Overview](/configuration/) for the full env-ref behaviour.
