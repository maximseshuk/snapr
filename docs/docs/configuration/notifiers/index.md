# Notifiers

`notifiers` is an array; each entry sends a message after a job runs. Every notifier toggles `onSuccess` / `onFailure` independently — fire on failures only, on every run, or on success.

## Common fields

| Field       | Type   | Required | Notes                          |
| ----------- | ------ | -------- | ------------------------------ |
| `type`      | enum   | yes      | `webhook`, `telegram`, `email` |
| `name`      | string | no       | label shown in the UI          |
| `onSuccess` | bool   | no       | fire on successful runs        |
| `onFailure` | bool   | no       | fire on failed runs            |

If both `onSuccess` and `onFailure` are `false`, the notifier is effectively disabled.

## Types

- [Webhook](/configuration/notifiers/webhook) — HTTP POST/PUT/etc. to any URL
- [Telegram](/configuration/notifiers/telegram) — Telegram bot message
- [Email](/configuration/notifiers/email) — SMTP email
