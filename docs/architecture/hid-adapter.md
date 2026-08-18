# HID Input Adapter

The Linux-only `source/hid` adapter reads keyboard-like events from a Linux input device and converts keyboard-like values terminated by Enter into `scanner.Input` values.

Application configuration uses `DEVICE_BRIDGE_HID_PATH`, for example `/dev/input/event2`. It is mutually exclusive with `DEVICE_BRIDGE_SCANNER_PATH`.

## Scope

- Reads Linux `input_event` records from an `io.ReadCloser`.
- Supports digit key codes and Enter termination.
- Supports US-layout letters, digits, punctuation, Shift, Caps Lock, keypad digits, Space, and Backspace.
- Ignores non-key and key-press events.
- Closes the device on context cancellation to interrupt blocking reads.
- Does not implement low-level USB HID descriptors, `hidraw`, or libusb transfers.

`uinput` can provide a virtual Linux input device for integration tests. Real USB HID protocol behavior still requires hardware passthrough or a USB gadget fixture.

The adapter does not interpret GS/FNC1 semantics. If the scanner is configured to replace GS with a keyboard sequence, that sequence is delivered as ordinary input characters/control data for higher-level processing.

The tagged Linux integration test creates a temporary `uinput` device when `/dev/uinput` is available and skips cleanly otherwise. The regular cross-platform test suite does not require `uinput`.
