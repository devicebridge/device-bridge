# Debian 12 Deployment

This guide covers the first alpha deployment on Debian 12 with either a Linux HID scanner or a serial-compatible scanner.

## Build

Build on the development machine for an amd64 Debian target:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -ldflags="-s -w" \
  -o device-bridge-linux-amd64 ./cmd/device-bridge
```

Verify the artifact:

```bash
file device-bridge-linux-amd64
```

Copy and install it on the target:

```bash
scp device-bridge-linux-amd64 root@TARGET:/tmp/device-bridge
ssh root@TARGET 'install -o root -g root -m 0755 /tmp/device-bridge /usr/local/bin/device-bridge'
```

The target architecture should be checked with `uname -m`; `x86_64` requires `GOARCH=amd64`.

## Service User

Create a restricted service account. Add `input` for HID devices and `dialout` for serial devices:

```bash
sudo useradd --system --home /var/lib/device-bridge \
  --shell /usr/sbin/nologin devicebridge
sudo install -d -o devicebridge -g devicebridge -m 0750 /var/lib/device-bridge
sudo usermod -aG input,dialout devicebridge
```

For a HID scanner, inspect permissions:

```bash
stat -c '%A %U %G %n' /dev/input/event2
```

For serial, inspect `/dev/ttyUSB*` or `/dev/ttyACM*` and ensure the device group is `dialout`.

## Configuration

Create `/etc/device-bridge/device-bridge.env`.

HID mode:

```dotenv
DEVICE_BRIDGE_HTTP_HOST=0.0.0.0
DEVICE_BRIDGE_HTTP_PORT=8080
DEVICE_BRIDGE_SOURCES=scanner-main
DEVICE_BRIDGE_HID_PATH=/dev/input/event2
```

Serial mode:

```dotenv
DEVICE_BRIDGE_HTTP_HOST=0.0.0.0
DEVICE_BRIDGE_HTTP_PORT=8080
DEVICE_BRIDGE_SOURCES=scanner-main
DEVICE_BRIDGE_SCANNER_PATH=/dev/ttyUSB0
DEVICE_BRIDGE_SCANNER_BAUD=9600
DEVICE_BRIDGE_SCANNER_RECONNECT_SECONDS=1
```

Do not set HID and serial paths together. For serial alpha testing, configure baud/parity/data bits/stop bits with the OS/device tooling before starting the service. The current application validates `DEVICE_BRIDGE_SCANNER_BAUD`, but does not yet apply all serial line parameters itself.

Protect the environment file:

```bash
sudo chown root:root /etc/device-bridge/device-bridge.env
sudo chmod 0640 /etc/device-bridge/device-bridge.env
```

## systemd

Create `/etc/systemd/system/device-bridge.service`:

```ini
[Unit]
Description=Device Bridge
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=devicebridge
Group=devicebridge
SupplementaryGroups=input dialout
EnvironmentFile=/etc/device-bridge/device-bridge.env
ExecStart=/usr/local/bin/device-bridge
Restart=on-failure
RestartSec=3s
TimeoutStopSec=10s

NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/device-bridge

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now device-bridge.service
```

Restart after configuration changes:

```bash
sudo systemctl restart device-bridge.service
```

## Quick Check

Check service state:

```bash
systemctl status device-bridge.service --no-pager
```

Check endpoints locally:

```bash
curl -i http://127.0.0.1:8080/healthz
curl -i http://127.0.0.1:8080/readyz
```

Both should return `200` while the application is running. Connect a WebSocket client to `ws://TARGET:8080/ws` and scan a test value.

## Diagnostics

Follow service logs:

```bash
sudo journalctl -u device-bridge.service -f
```

Show recent logs:

```bash
sudo journalctl -u device-bridge.service -n 100 --no-pager
```

Show service exit and restart information:

```bash
sudo systemctl show device-bridge.service \
  -p ActiveState -p SubState -p ExecMainStatus -p NRestarts
```

HID discovery:

```bash
lsusb
evtest --list
cat /proc/bus/input/devices
```

Serial discovery:

```bash
ls -l /dev/ttyUSB* /dev/ttyACM* 2>/dev/null
stty -F /dev/ttyUSB0 -a
```

Do not run `evtest` against the scanner while Device Bridge is reading the same event device; both consumers can compete for diagnostics. Stop the service first or use a separate test window carefully.

## Stable Device Paths

`/dev/input/eventN` and `/dev/ttyUSB0` can change after reboot or reconnect. For a long-running installation, create a udev symlink based on the actual vendor/product/serial attributes, then use the symlink in `DEVICE_BRIDGE_HID_PATH` or `DEVICE_BRIDGE_SCANNER_PATH`.

Inspect attributes:

```bash
udevadm info --query=all --name=/dev/input/event2
udevadm info --query=all --name=/dev/ttyUSB0
```

The exact udev rule must be built from the target device's attributes.

## Stop and Uninstall

```bash
sudo systemctl disable --now device-bridge.service
sudo rm /etc/systemd/system/device-bridge.service
sudo systemctl daemon-reload
sudo rm /usr/local/bin/device-bridge
sudo rm /etc/device-bridge/device-bridge.env
```

Remove the service user only if it is no longer needed:

```bash
sudo userdel devicebridge
```
