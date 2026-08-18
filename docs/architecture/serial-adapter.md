# Serial Adapter

`scanner.SerialAdapter` is the transport boundary between a serial-compatible byte stream and the scanner source.

## Contract

- The port implements `io.ReadCloser`.
- Input is line-oriented.
- Both `LF` and `CRLF` terminate a value; the terminator is removed.
- A final non-empty value before `EOF` is delivered.
- Port errors are returned to the source runtime.
- Context cancellation closes the port so a blocking read can be interrupted.
- The adapter does not open device paths itself. Opening/configuring a serial device belongs to the application/device integration layer.

This keeps unit tests independent from serial hardware, PTYs, `socat`, and operating-system-specific device APIs.
