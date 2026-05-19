# Permissions

`server.permissions` controls what authenticated users can do in the UI.

## Fields

| Field                 | Type | Default | Notes                                               |
| --------------------- | ---- | ------- | --------------------------------------------------- |
| `allowBackupDownload` | bool | `true`  | enable archive download buttons                     |
| `allowManualRun`      | bool | `true`  | enable on-demand "Run now" for jobs                 |
| `showConfig`          | bool | `true`  | render the parsed config in the UI (secrets masked) |

## Example

```yaml
server:
  permissions:
    allowBackupDownload: false
    allowManualRun: true
    showConfig: false
```
