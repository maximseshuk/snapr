# Webhook

`type: webhook` — sends an HTTP request with the run summary in the body.

## Fields

| Field     | Type         | Required | Notes                                      |
| --------- | ------------ | -------- | ------------------------------------------ |
| `type`    | `webhook`    | yes      |                                            |
| `url`     | string       | yes      |                                            |
| `method`  | string       | no       | `POST`, `PUT`, etc. Default `POST`.        |
| `headers` | map<str,str> | no       | extra HTTP headers (use `env:` for tokens) |

Plus the [common fields](/configuration/notifiers/) (`name`, `onSuccess`, `onFailure`).

## Example

```yaml
notifiers:
  - type: webhook
    name: ops
    url: https://hooks.example.com/services/abc123
    method: POST
    headers:
      Authorization: env:WEBHOOK_TOKEN
      X-App: snapr
    onSuccess: true
    onFailure: true
```
