# Overview

snapr is configured by a single YAML file.

## File location

Resolution order:

1. `-config <path>` CLI flag
2. `SNAPR_CONFIG_FILE` environment variable
3. `./snapr.yaml`
4. `~/.config/snapr/snapr.yaml`
5. `~/snapr.yaml`
6. `/etc/snapr/snapr.yaml`

The first match wins.

## Top-level shape

```yaml
logs: { ... }
server: { ... }
jobs:
  - { ... }
```

| Field    | Type   | Required | Notes                                                                                 |
| -------- | ------ | -------- | ------------------------------------------------------------------------------------- |
| `logs`   | object | no       | on-disk log files and rotation — see [Logs](/configuration/logs).                     |
| `server` | object | no       | HTTP listener and web UI — see [Server](/configuration/server/). Defaults to enabled. |
| `jobs`   | array  | yes      | at least one job — see [Jobs](/configuration/jobs)                                    |

## Environment references

Any string value may use the prefix `env:` to read its value from an environment variable at load time. Use this to keep secrets out of the YAML file.

```yaml
server:
  secret: env:SNAPR_JWT_SECRET

jobs:
  - name: postgres
    sources:
      - type: postgresql
        password: env:PG_PASSWORD
```

If the referenced variable is not set, snapr fails to start.

## Applying changes

Config is loaded once at startup. To apply edits, restart snapr. The previous configuration stays active until the new process loads successfully — a malformed file fails the restart with the error logged.
