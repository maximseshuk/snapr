# SQLite

`type: sqlite` — back up a single SQLite database file by running `sqlite3 <file> .dump` and writing the resulting SQL to the archive.

## Prerequisites

- `sqlite3` must be installed on the snapr host and available in `PATH` (Debian/Ubuntu: `sqlite3`).
- snapr needs read access to the database file. SQLite uses file locks during the dump, so backups taken while the application writes heavily may briefly fail — retry on the next schedule.

## Fields

| Field  | Type     | Required | Notes                              |
| ------ | -------- | -------- | ---------------------------------- |
| `type` | `sqlite` | yes      |                                    |
| `path` | string   | yes      | path to the `.db` / `.sqlite` file |

## Example

```yaml
sources:
  - type: sqlite
    path: /var/lib/app/app.db
```

## Notes

- The output is a `.sql` text dump named after the source file (e.g. `app.sql`). Restore with `sqlite3 new.db < app.sql`.
- For binary copies (faster restore, larger size) use a [local source](/configuration/sources/local) pointed at the database directory while the application is stopped.
