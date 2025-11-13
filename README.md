# SP Monitor — Lightweight Service & Port Dashboard

HTML dashboard showing status of local services via:

- TCP port checks
- systemd unit state (Linux)
- Windows services & processes (SC / tasklist / PowerShell)

Includes optional start/stop controls, auth, JSON status export, and action/status change logging.

## Features

- Unified view of mixed services (ports, systemd, Windows service/process, run-path executables)
- Automatic status refresh & JSON export (`status.json` by default)
- Start / Stop controls per service (granular flags)
- Simple cookie session auth
- CSV action & status change log with size limiting
- Portable (Linux & Windows)
- Minimal dependencies (standard Go only)

## Quick Start

```bash
go run .
# then visit http://localhost:7337
```

Build:

```bash
go build -o port-monitor
```

## Configuration Files

- `config.json` — core settings (see `docs/config-example.json`)
- `services.json` — service list (see `docs/services-example.json`)

Minimal `config.json`:

```json
{ "port": 7337 }
```

Supported `config.json` keys:

| Key | Description | Default |
| --- | ----------- | ------- |
| `port` | HTTP listen port | required |
| `services_file` | Path to services list | `services.json` |
| `web_dir` | Static assets directory | `web` |
| `template_file` | HTML template path | `web/index.html` |
| `admin_login` / `admin_password` | Credentials for UI/API | empty (auth disabled) |
| `log_file` | CSV log path | `log.csv` |
| `log_max_bytes` | Max log size (truncate) | unset |

`services.json` service fields:

| Field | Purpose |
| ----- | ------- |
| `name` | Display name (card title) |
| `port` | TCP port to probe (optional if using service/systemd) |
| `link` / `image` | Optional URL + icon for UI |
| `show_port` | Show port number on card |
| `service_name` | Windows service name OR process/exe identifier (also used for Linux if `systemd_name` absent) |
| `systemd_name` | Linux systemd unit name (e.g. `my-app.service`) |
| `controls` | Enable any controls for this service |
| `controls_run` / `controls_shut` | Enable start / stop respectively |
| `run_path` | Direct executable/script to start (bypasses service manager) |
| `run_env` | Extra env vars when starting `run_path` |

## Environment Variables

| Variable | Default | Notes |
| -------- | ------- | ----- |
| `EXPORT_PATH` | `.` | Directory for status export |
| `EXPORT_NAME` | `status.json` | Export file name |
| `IMPORT_PATH` | `EXPORT_PATH` | Read path for rendering (allows sharing) |
| `IMPORT_NAME` | `EXPORT_NAME` | Read file name |
| `STATUS_INTERVAL` | `5s` | Refresh interval (duration) |
| `PORT_DIAL_TIMEOUT` | `200ms` | TCP dial timeout per check |

Status is written periodically to `EXPORT_PATH/EXPORT_NAME` and read from `IMPORT_PATH/IMPORT_NAME` (can differ to consume external status file).

## API

All JSON responses; authentication via `POST /api/login` sets `session` cookie.

| Endpoint | Method | Description |
| -------- | ------ | ----------- |
| `/api/login` | POST | Body: `{"login":"...","password":"..."}`; sets session |
| `/api/logout` | POST | Clear session |
| `/api/me` | GET | Returns `{ loggedIn, user }` |
| `/api/logs?limit=N` | GET | Last N log lines (excludes header); auth required |
| `/api/service/start` | POST | Body contains identifier (`name` / `service_name` / `systemd_name` / `port`) |
| `/api/service/stop` | POST | Same identifier schema |

Service action requires: authenticated user + `controls=true` and respective `controls_run` / `controls_shut`.

## Logging

CSV (`log_file`) header:

```text
timestamp,user,ip,action,name,service_name,systemd_name,port,result
```

Entries prepend (newest at top). Actions logged:

- Status transitions (`up` / `down`)
- Start / Stop attempts (result `ok` or error message)

Size trimming if `log_max_bytes` set.

## Platform Notes

- Linux: status via `systemctl is-active`, port via `ss -tuln` fallback TCP dial; control via `systemctl start/stop` or `run_path`.
- Windows: status via `sc query`, `tasklist`, PowerShell fallback; control via `sc start/stop`, `Start-Process` for executables, `taskkill` for stop.

## Web UI

- Template: `web/index.html` (Go `html/template`)
- Styles: `web/styles.css`
- Active services sorted first, then inactive

## Security

In-memory sessions only. Use reverse proxy + HTTPS. Set strong `admin_password`. Limit exposed control endpoints to trusted networks.

## Build / Install

```bash
go build -o port-monitor
./port-monitor
```

Systemd integration / installer script (if added) would reside in `docs/` (not included in this summary).

---

Minimal, dependency-free: ideal for small self-hosted dashboards.
