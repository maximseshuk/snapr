# Redis

`type: redis` — produces an RDB snapshot. Two modes:

1. **Live dump (default)** — runs `redis-cli --rdb <out>` against a running server. Redis triggers a background save and streams the resulting RDB to snapr.
2. **File copy** — when `path` is set, snapr simply copies the existing RDB file from disk. No server connection is made.

## Prerequisites

- For mode 1: `redis-cli` must be installed on the snapr host and available in `PATH` (Debian/Ubuntu: `redis-tools`).
- For mode 2: snapr just needs read access to the RDB file — no client tools required.
- The configured user (when ACLs are enabled) needs at least the `~* +@read +bgsave +replconf +psync +ping` permissions, or simply the predefined `~* +@all` role for a dedicated backup user.

## Fields

| Field      | Type    | Required | Notes                                                                       |
| ---------- | ------- | -------- | --------------------------------------------------------------------------- |
| `type`     | `redis` | yes      |                                                                             |
| `path`     | string  | no       | absolute path to an RDB file. When set, snapr copies it instead of dumping. |
| `host`     | string  | yes\*    | required unless `socket` or `path` is set. Defaults to `127.0.0.1`.         |
| `port`     | int     | no       | default `6379`                                                              |
| `socket`   | string  | no       | UNIX socket path                                                            |
| `username` | string  | no       | Redis 6+ ACL username (passed as `--user`)                                  |
| `password` | string  | no       | passed via `REDISCLI_AUTH` env var, never on cmdline. Use `env:`.           |

## Example — live dump

```yaml
sources:
  - type: redis
    host: cache.internal
    port: 6379
    username: backup
    password: env:REDIS_PASSWORD
```

## Example — file copy

```yaml
sources:
  - type: redis
    path: /var/lib/redis/dump.rdb
```

## Notes

- Live dump triggers a save on the server side. On large datasets the snapshot fork can briefly impact memory and latency.
- The output filename is always `dump.rdb` inside the job's working directory.
