# Log limits

`server.logLimits` controls how many lines the API returns in the **initial** tail request that backs the log viewer. It's the default value for the `?tail=N` query parameter on `/api/v1/logs/system` and `/api/v1/jobs/{name}/logs`. Once the UI has the initial slice it opens an SSE stream and appends every new line as it lands — there is no client-side cap.

This block does **not** size the on-disk log files themselves — see [Logs](/configuration/logs) for path, rotation, and per-job/system toggles.

## Fields

| Field        | Type | Default | Notes                                  |
| ------------ | ---- | ------- | -------------------------------------- |
| `jobLogs`    | int  | `10000` | initial tail length per job log viewer |
| `systemLogs` | int  | `10000` | initial tail length for the system log |

A client may request fewer lines via `?tail=N`. Requests above 50 000 are clamped server-side to keep a single response from reading millions of lines off disk in one go.

## Example

```yaml
server:
  logLimits:
    jobLogs: 5000
    systemLogs: 20000
```

## Edge cases

- Setting either field to `0` falls back to the default (10 000). Use a small positive number to actually shrink the initial tail.
- The hard server-side cap of 50 000 is independent of these defaults; even an explicit `?tail=999999` is clamped to 50 000.
