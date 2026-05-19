# Authentication

`server.auth` enables a login flow for the web UI and protects API endpoints. JWT-based.

## Fields

| Field             | Type   | Default | Notes                                      |
| ----------------- | ------ | ------- | ------------------------------------------ |
| `enabled`         | bool   | `false` | turns auth on                              |
| `username`        | string | —       | required when `enabled: true`              |
| `password`        | string | —       | required when `enabled: true`. Use `env:`. |
| `tokenExpiration` | int    | —       | minutes before the JWT expires             |
| `cookies`         | object | —       | see below                                  |

## `cookies`

| Field      | Type   | Default | Notes                         |
| ---------- | ------ | ------- | ----------------------------- |
| `secure`   | bool   | `false` | sets the `Secure` cookie flag |
| `sameSite` | string | —       | `lax`, `strict`, or `none`    |
| `domain`   | string | —       | cookie `Domain` attribute     |

## Example

```yaml
server:
  secret: env:SNAPR_JWT_SECRET
  auth:
    enabled: true
    username: admin
    password: env:SNAPR_ADMIN_PASSWORD
    tokenExpiration: 1440
    cookies:
      secure: true
      sameSite: strict
      domain: snapr.example.com
```

## Notes

- Set `server.secret` whenever auth is enabled. A weak or default secret allows token forgery.
- For HTTPS deployments behind a reverse proxy set `cookies.secure: true`.
