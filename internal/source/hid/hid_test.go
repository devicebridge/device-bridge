//go:build linux

package hid

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

func TestAdapterTranslatesDigits(t *testing.T) {
	data := bytes.NewBuffer(nil)
	writeKey(data, 2)
	writeKey(data, 3)
	writeKey(data, keyEnter)
	a := New(nopCloser{Buffer: data}, 1)

	done := make(chan error, 1)
	go func() { done <- a.Run(context.Background()) }()

	select {
	case input := <-a.Input():
		if input.Value != "12" {
			t.Fatalf("expected 12, got %q", input.Value)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for HID input")
	}
	a.Close()
	<-done
}

func TestKeyCodeCharacterUSLayout(t *testing.T) {
	cases := []struct {
		code  uint16
		state keyboardState
		want  byte
	}{
		{30, keyboardState{}, 'a'},
		{30, keyboardState{shift: true}, 'A'},
		{30, keyboardState{caps: true}, 'A'},
		{2, keyboardState{shift: true}, '!'},
		{12, keyboardState{shift: true}, '_'},
		{53, keyboardState{shift: true}, '?'},
	}
	for _, test := range cases {
		got, ok := keyCodeCharacter(test.code, test.state)
		if !ok || got != test.want {
			t.Fatalf("code %d state %+v: got %q/%v, want %q", test.code, test.state, got, ok, test.want)
		}
	}
}

func TestAdapterCancellationClosesDevice(t *testing.T) {
	device := &blockingDevice{done: make(chan struct{})}
	a := New(device, 1)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- a.Run(ctx) }()

	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("HID adapter did not stop after cancellation")
	}
}

type nopCloser struct{ *bytes.Buffer }

func (n nopCloser) Close() error { return nil }

func writeKey(buf *bytes.Buffer, code uint16) {
	event := make([]byte, inputEventSize)
	binary.LittleEndian.PutUint16(event[16:18], evKey)
	binary.LittleEndian.PutUint16(event[18:20], code)
	binary.LittleEndian.PutUint32(event[20:24], keyRelease)
	buf.Write(event)
}

type blockingDevice struct {
	done chan struct{}
	once sync.Once
}

func (d *blockingDevice) Read([]byte) (int, error) {
	<-d.done
	return 0, io.ErrClosedPipe
}

func (d *blockingDevice) Close() error {
	d.once.Do(func() { close(d.done) })
	return nil
}
