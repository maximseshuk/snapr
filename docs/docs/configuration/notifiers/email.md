# Email

`type: email` — sends an SMTP message to one or more recipients.

## Fields

| Field      | Type     | Required | Notes                                                                                                   |
| ---------- | -------- | -------- | ------------------------------------------------------------------------------------------------------- |
| `type`     | `email`  | yes      |                                                                                                         |
| `smtpHost` | string   | yes      | SMTP server hostname                                                                                    |
| `smtpPort` | int      | yes      | usually `25`, `465`, or `587`                                                                           |
| `smtpUser` | string   | no       | SMTP auth user (use `env:`)                                                                             |
| `smtpPass` | string   | no       | SMTP auth password (use `env:`)                                                                         |
| `from`     | string   | yes      | `From:` address                                                                                         |
| `to`       | string[] | yes      | one or more recipients                                                                                  |
| `useTLS`   | bool     | no       | dial the SMTP server over implicit TLS (typically port `465`). Leave `false` for plaintext or STARTTLS. |

Plus the [common fields](/configuration/notifiers/).

## Example

```yaml
notifiers:
  - type: email
    name: ops-mail
    smtpHost: smtp.example.com
    smtpPort: 587
    smtpUser: alerts@example.com
    smtpPass: env:SMTP_PASSWORD
    from: alerts@example.com
    to:
      - oncall@example.com
      - sre@example.com
    onFailure: true
```

## TLS modes

- **Plain → STARTTLS upgrade (port `587`)**: leave `useTLS: false`. snapr negotiates STARTTLS automatically when the server advertises it.
- **Implicit TLS (port `465`)**: set `useTLS: true`. snapr opens a TLS connection from the start.
- **Plaintext**: leave `useTLS: false` and connect to a server that does not advertise STARTTLS. Not recommended.

The recipient's `From:` header is taken verbatim from `from`. `smtpUser` defaults to no auth — provide both `smtpUser` and `smtpPass` only when the server requires `LOGIN`/`PLAIN` auth.
