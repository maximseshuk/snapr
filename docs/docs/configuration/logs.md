---
title: Logs
---

# Logs

Top-level `logs:` block. Controls how snapr writes log files to disk.

snapr always emits two streams in parallel:

- **stdout** — human-readable lines with ANSI colour, never disabled, never written to disk by snapr itself
- **files** — JSON-line events on disk, rotated by snapr; this is what the API and UI read from

Web UI panels (System log, per-job log) read from these files. Disabling a file disables the corresponding UI panel — snapr returns 503 from the matching API endpoint and the UI shows a "log is disabled" placeholder.

## Layout on disk

When both flags are on (the default), the layout is:

```
<logs.path>/
  snapr.log                 # all events
  jobs/
    <job-name>.log          # events tagged with job=<name>
```

Job-name characters outside `[A-Za-z0-9._-]` collapse to `_` to keep filenames safe.

## Fields

| Field        | Type   | Default  | Notes                                                                  |
| ------------ | ------ | -------- | ---------------------------------------------------------------------- |
| `path`       | string | `./logs` | directory; created on start if missing                                 |
| `system`     | bool   | `true`   | write the combined `snapr.log`                                         |
| `perJob`     | bool   | `true`   | write `jobs/<name>.log` per job                                        |
| `maxSizeMB`  | int    | `100`    | rotate when a file reaches this size; `0` disables size-based rotation |
| `maxBackups` | int    | `7`      | how many rotated files to keep; `0` keeps all                          |
| `maxAgeDays` | int    | `30`     | delete rotated files older than this; `0` disables age-based pruning   |
| `compress`   | bool   | `true`   | gzip rotated files                                                     |

`path` is resolved relative to the process working directory. Use an absolute path in production (e.g. `/var/log/snapr`).

## Example

```yaml
logs:
  path: /var/log/snapr
  system: true
  perJob: true
  maxSizeMB: 100
  maxBackups: 7
  maxAgeDays: 30
  compress: true
```

## Disabling files

Set `system: false` and/or `perJob: false`. With both off, no file sink is created and the API endpoints return:

```
503 Service Unavailable
{"error":"System log is disabled (logs.system=false)"}
```

The UI hides the log toolbar and shows a placeholder explaining which flag turned the panel off.

stdout is unaffected — `journalctl`, `docker logs`, or your container runtime still see everything.

## Edge cases

- **Lumberjack rotation is in-process.** A second snapr process pointed at the same `path` will fight for rotation; don't run two instances against one log directory.
- **No external rotation needed.** Don't combine snapr's rotation with `logrotate(8)`/`copytruncate` — set snapr's `maxSizeMB: 0` if you'd rather an external tool own rotation.
- **Disk full.** Lumberjack swallows write errors silently to keep the process alive; the same event still hits stdout, so external collectors keep working.
- **Removed jobs.** When a job is deleted from the config, its `jobs/<name>.log` is left in place. Clean up manually if needed.
- **Renamed jobs.** A rename starts a new file under the new name; the old file is not migrated.
- **`server.enabled: false` (scheduler-only mode).** File logs still work the same way; only the API endpoints that read them are gone.
- **Permissions.** snapr creates `path` with `0755` and files with lumberjack's default (`0600`). Make sure the process user can write there — failure to create the directory aborts startup.
