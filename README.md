# Port Monitor Service

![Изображение интерфейса SP-Monitor](https://files.oleg.fans/2Kg87j.png)

This service displays an HTML dashboard showing the status of configured services, validated by checking if ports are in use or if systemd services are active.

## Configuration

The service is configured using two JSON files:

1. `config.json`:
```json
{
  "port": 7337
}
```
- `port`: The port where the web interface will be available

2. `services.json`:
```json
{
  "services": [
    {
      "port": 7331,
      "name": "Main page",
      "link": "https://oleg.fans",
      "image": "https://oleg.fans/favicon.ico",
      "show_port": false
    },
    {
      "name": "Gala Music Bot",
      "systemd_name": "GALA-tgmusicbot",
      "link": "https://t.me/Klaus_House",
      "image": "https://files.oleg.fans/p8S2bg.jpg"
    }
  ]
}
```

Each service can have these properties:
- `port`: Port number to check (optional if using systemd_name)
- `name`: Display name for the service
- `link`: Optional URL to link to
- `image`: Optional URL to service icon/favicon
- `show_port`: Whether to display the port number on the dashboard
- `systemd_name`: Optional systemd service name to check instead of port (for systemd services)

## Features

- Monitor both port-based services and systemd services
- Color-coded status indicators (green for running, red for stopped)
- Dashboard with service icons, names, and links
- Responsive web design
- Automatic sorting: active services appear first
- Customizable service display with optional port visibility

## Usage

1. Make sure Go is installed
2. Run the service directly:
   ```
   go run main.go
   ```
3. Access the web interface at `http://localhost:7337`

## Building

To build an executable:

```
go build -o port-monitor
```

Then run it with:

```
./port-monitor
```

## Installing as a Systemd Service

To install and run the service as a systemd service:

1. Build the executable:
   ```
   go build -o port-monitor
   ```

2. Run the install script from the docs directory:
   ```bash
   cd docs
   sudo ./install.sh
   ```

3. Check the service status:
   ```bash
   sudo systemctl status port-monitor.service
   ```

4. Stop the service if needed:
   ```bash
   sudo systemctl stop port-monitor.service
   ```

The service will be available at `http://localhost:7337`.
