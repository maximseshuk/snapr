# UI

`server.ui` toggles the bundled web UI (the SPA assets and HTML fallback). The HTTP listener itself is configured by the surrounding [`server`](/configuration/server/) block.

## Fields

| Field     | Type | Default | Notes                                           |
| --------- | ---- | ------- | ----------------------------------------------- |
| `enabled` | bool | `true`  | serve the bundled web UI alongside the JSON API |

Setting `server.ui.enabled: true` while `server.enabled: false` is rejected at startup — the UI cannot be served without a listener.

## API-only mode

Useful when snapr is driven by an external dashboard or automation and the bundled UI would only be dead weight.

```yaml
server:
  enabled: true
  address: '0.0.0.0:8080'
  secret: env:SNAPR_JWT_SECRET
  auth:
    enabled: true
    username: admin
    password: env:SNAPR_ADMIN_PASSWORD
  ui:
    enabled: false

jobs:
  - name: nightly
    schedule: '0 2 * * *'
    # ...
```

In this mode `/api/*` endpoints behave as usual; requests to other paths return `404`.
