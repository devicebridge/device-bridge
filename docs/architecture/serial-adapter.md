# Serial Adapter

`scanner.SerialAdapter` is the transport boundary between a serial-compatible byte stream and the scanner source.

## Contract

- The port implements `io.ReadCloser`.
- Input is line-oriented.
- Both `LF` and `CRLF` terminate a value; the terminator is removed.
- A final non-empty value before `EOF` is delivered.
- Port errors are returned to the source runtime.
- Context cancellation closes the port so a blocking read can be interrupted.
- `ReconnectingAdapter` retries opening and reading after recoverable port failures using a cancellable delay.
- The adapter does not open device paths itself. Opening/configuring a serial device belongs to the application/device integration layer.

This keeps unit tests independent from serial hardware, PTYs, `socat`, and operating-system-specific device APIs.

## Manual PTY check

On a Linux development VM, create two linked PTYs:

```bash
socat -d -d \
  pty,raw,echo=0,link=/tmp/device-bridge-app \
  pty,raw,echo=0,link=/tmp/device-bridge-injector
```

Run the application with `DEVICE_BRIDGE_SOURCES=scanner-main` and `DEVICE_BRIDGE_SCANNER_PATH=/tmp/device-bridge-app`. In a second terminal send a line:

```bash
printf '1234567890123\n' > /tmp/device-bridge-injector
```

The Linux CI integration test uses the same PTY model with a temporary `socat` process. It is isolated behind the `integration` build tag, so the cross-platform unit test suite does not require Linux PTYs or `socat`.
