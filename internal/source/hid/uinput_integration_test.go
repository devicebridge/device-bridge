//go:build linux && integration

package hid

import (
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

const (
	uiSetEvBit   = 1074025828
	uiSetKeyBit  = 1074025829
	uiDevCreate  = 21761
	uiDevDestroy = 21762
	uiDevName    = "device-bridge-uinput"
)

func TestAdapterWithUInput(t *testing.T) {
	device, err := os.OpenFile("/dev/uinput", os.O_WRONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		t.Skipf("/dev/uinput is unavailable: %v", err)
	}
	defer device.Close()

	if err := uinputIoctl(device, uiSetEvBit, evKey); err != nil {
		t.Skipf("cannot configure /dev/uinput: %v", err)
	}
	for _, code := range []uint16{2, 3, keyEnter} {
		if err := uinputIoctl(device, uiSetKeyBit, uintptr(code)); err != nil {
			t.Skipf("cannot configure key %d: %v", code, err)
		}
	}
	var descriptor uinputUserDev
	copy(descriptor.Name[:], uiDevName)
	descriptor.ID.Bustype = 3
	descriptor.ID.Vendor = 1
	descriptor.ID.Product = 1
	descriptor.ID.Version = 1
	if _, err := device.Write(descriptor.bytes()); err != nil {
		t.Skipf("cannot write uinput descriptor: %v", err)
	}
	if err := uinputIoctl(device, uiDevCreate, 0); err != nil {
		t.Skipf("cannot create uinput device: %v", err)
	}
	defer uinputIoctl(device, uiDevDestroy, 0)

	eventPath := waitForEventDevice(t)
	input, err := os.OpenFile(eventPath, os.O_RDONLY, 0)
	if err != nil {
		if os.IsPermission(err) {
			t.Skipf("cannot read uinput event device %s: %v", eventPath, err)
		}
		t.Fatal(err)
	}
	adapter := New(input, 1)
	done := make(chan error, 1)
	go func() { done <- adapter.Run(context.Background()) }()

	for _, code := range []uint16{2, 3, keyEnter} {
		if err := writeInputEvent(device, code, keyRelease); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case value := <-adapter.Input():
		if value.Value != "12" {
			t.Fatalf("expected 12, got %q", value.Value)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for uinput event")
	}
	adapter.Close()
	input.Close()
	<-done
}

type inputID struct{ Bustype, Vendor, Product, Version uint16 }
type uinputUserDev struct {
	Name         [80]byte
	ID           inputID
	FFEffectsMax uint32
	Absmax       [64]int32
	Absmin       [64]int32
	Absfuzz      [64]int32
	Absflat      [64]int32
}

func (u *uinputUserDev) bytes() []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(u)), int(unsafe.Sizeof(*u)))
}

func uinputIoctl(file *os.File, request uintptr, value uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), request, value)
	if errno != 0 {
		return errno
	}
	return nil
}

func writeInputEvent(file *os.File, code, value uint16) error {
	event := make([]byte, inputEventSize)
	binary.LittleEndian.PutUint16(event[16:18], evKey)
	binary.LittleEndian.PutUint16(event[18:20], code)
	binary.LittleEndian.PutUint32(event[20:24], uint32(value))
	_, err := file.Write(event)
	return err
}

func waitForEventDevice(t *testing.T) string {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		paths, _ := filepath.Glob("/dev/input/event*")
		for _, path := range paths {
			namePath := filepath.Join("/sys/class/input", filepath.Base(path), "device", "name")
			name, err := os.ReadFile(namePath)
			if err != nil {
				continue
			}
			if string(name) == uiDevName+"\n" {
				if info, err := os.Stat(path); err == nil && info.Mode()&os.ModeCharDevice != 0 {
					return path
				}
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("uinput event device was not created")
	return ""
}
