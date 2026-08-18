# HID Input Adapter

The Linux-only `source/hid` adapter reads keyboard-like events from a Linux input device and converts digit key releases terminated by Enter into `scanner.Input` values.

## Scope

- Reads Linux `input_event` records from an `io.ReadCloser`.
- Supports digit key codes and Enter termination.
- Ignores non-key and key-press events.
- Closes the device on context cancellation to interrupt blocking reads.
- Does not implement low-level USB HID descriptors, `hidraw`, or libusb transfers.

`uinput` can provide a virtual Linux input device for integration tests. Real USB HID protocol behavior still requires hardware passthrough or a USB gadget fixture.

The tagged Linux integration test creates a temporary `uinput` device when `/dev/uinput` is available and skips cleanly otherwise. The regular cross-platform test suite does not require `uinput`.
