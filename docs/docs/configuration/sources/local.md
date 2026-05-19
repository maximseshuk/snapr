# Local files

`type: local` — back up files and directories on the host filesystem.

## Fields

| Field      | Type     | Required | Notes                                                                             |
| ---------- | -------- | -------- | --------------------------------------------------------------------------------- |
| `type`     | `local`  | yes      |                                                                                   |
| `path`     | string   | yes      | absolute or relative path to the directory                                        |
| `excludes` | string[] | no       | glob patterns matched against paths relative to `path` (skipped from the archive) |

## Notes

- When `excludes` is empty, snapr symlinks the source path into the working directory (no copy). With `excludes`, snapr walks the tree and copies file-by-file.
- Patterns are matched against each entry's path relative to `path`. Supported forms:
  - `*.tmp` — glob (Go `filepath.Match` syntax) against the path or just the filename.
  - `temp/` — trailing slash means "anything under this directory".
  - `**/cache/**` — match a named directory at any depth.
  - `node_modules` — bare literal also matches by directory or filename.
- No external tools required.

## Example

```yaml
sources:
  - type: local
    path: /var/data
    excludes:
      - '*.tmp'
      - '*.log'
      - 'node_modules'
```
