# PostgreSQL

`type: postgresql` — uses `pg_dump` to produce a logical dump of one database.

## Prerequisites

- `pg_dump` must be installed on the snapr host and available in `PATH`.
- The major version of `pg_dump` should match the PostgreSQL **server** version. A newer client can dump older servers, but an older client against a newer server will fail.
- Install via the official PostgreSQL apt repo or your distro's `postgresql-client-<version>` package.
- The configured user needs at least `CONNECT` on the database and `SELECT` on every table being dumped (typically a dedicated read-only role).

## Fields

| Field           | Type         | Required | Notes                                                            |
| --------------- | ------------ | -------- | ---------------------------------------------------------------- |
| `type`          | `postgresql` | yes      |                                                                  |
| `host`          | string       | yes      |                                                                  |
| `port`          | int          | no       | default `5432`                                                   |
| `username`      | string       | yes      |                                                                  |
| `password`      | string       | no       | passed via `PGPASSWORD` env var, never on cmdline. Use `env:`    |
| `database`      | string       | yes      | database name                                                    |
| `excludeTables` | string[]     | no       | passed as `--exclude-table` flags                                |
| `extraParams`   | map<str,str> | no       | extra `pg_dump` flags (e.g. `schema-only: ""`, `format: custom`) |

## Example

```yaml
sources:
  - type: postgresql
    host: db.internal
    port: 5432
    username: postgres
    password: env:PG_PASSWORD
    database: app
    excludeTables:
      - audit_log
    extraParams:
      no-owner: ''
      format: plain
```

## Notes

- `extraParams` keys are prefixed with `--`. Use an empty string value for boolean flags.
- The password is forwarded through `PGPASSWORD` so it never appears in the process listing.
