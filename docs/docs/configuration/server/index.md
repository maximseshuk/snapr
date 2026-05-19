# Server

Top-level `server:` block. Controls the HTTP listener — JSON API, the bundled web UI, authentication, permissions, and runtime limits.

The HTTP listener has two sub-blocks of its own:

- [`server.ui`](/configuration/server/ui) — toggle the bundled UI (SPA assets and HTML fallback) on or off
- [`server.enabled: false`](#scheduler-only-mode) — drop the listener entirely

When `server.enabled: false` and `server.ui.enabled: true` snapr refuses to start — the UI cannot be served without a listener.

## Fields

| Field             | Type     | Default        | Notes                                                              |
| ----------------- | -------- | -------------- | ------------------------------------------------------------------ |
| `enabled`         | bool     | `true`         | turn the HTTP listener off to run snapr as a scheduler-only daemon |
| `address`         | string   | `0.0.0.0:8080` | listen address (`host:port`). Required when `enabled: true`.       |
| `secret`          | string   | —              | JWT signing secret. Use `env:` for production.                     |
| `defaultLanguage` | `en\|ru` | `en`           | UI default language                                                |
| `auth`            | object   | —              | see [Authentication](/configuration/server/auth)                   |
| `logLimits`       | object   | —              | see [Log limits](/configuration/server/log-limits)                 |
| `permissions`     | object   | —              | see [Permissions](/configuration/server/permissions)               |
| `ui`              | object   | —              | see [UI](/configuration/server/ui)                                 |

## Scheduler-only mode

Set `server.enabled: false` to drop the HTTP listener entirely. Jobs still execute on their cron schedules and write logs to the host. Useful for:

- minimal sidecars in a controlled environment
- machines where the listener would be unreachable anyway
- locking out manual runs and downloads at the binary level

```yaml
server:
  enabled: false

jobs:
  - name: nightly
    schedule: '0 2 * * *'
    # ...
```

## Example

```yaml
server:
  enabled: true
  address: '0.0.0.0:8080'
  secret: env:SNAPR_JWT_SECRET
  defaultLanguage: en
  auth:
    enabled: true
    username: admin
    password: env:SNAPR_ADMIN_PASSWORD
    tokenExpiration: 60
  permissions:
    allowBackupDownload: true
    allowManualRun: true
    showConfig: true
  ui:
    enabled: true
```
