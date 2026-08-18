//go:build linux

package hid

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"sync"

	"github.com/devicebridge/device-bridge/internal/source/scanner"
)

const (
	inputEventSize = 24
	evKey          = 1
	evSyn          = 0
	keyRelease     = 0
	keyEnter       = 28
)

var ErrUnsupportedEvent = errors.New("unsupported Linux input event")

type Adapter struct {
	device scannerPort
	input  *scanner.ChannelAdapter
	once   sync.Once
}

type scannerPort interface {
	io.Reader
	io.Closer
}

func New(device scannerPort, buffer int) *Adapter {
	if device == nil {
		panic("hid device cannot be nil")
	}
	return &Adapter{device: device, input: scanner.NewChannelAdapter(buffer)}
}

func (a *Adapter) Input() <-chan scanner.Input { return a.input.Input() }

// Run translates keyboard-like HID events into scanner values terminated by Enter.
func (a *Adapter) Run(ctx context.Context) error {
	defer a.input.Close()
	var value []byte
	readDone := make(chan readResult, 1)
	go a.readLoop(readDone)
	for {
		var event []byte
		select {
		case <-ctx.Done():
			a.Close()
			return ctx.Err()
		case result := <-readDone:
			if result.err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				return result.err
			}
			event = result.event
		}
		typeID := binary.LittleEndian.Uint16(event[16:18])
		code := binary.LittleEndian.Uint16(event[18:20])
		keyValue := binary.LittleEndian.Uint32(event[20:24])
		if typeID != evKey || keyValue != keyRelease {
			continue
		}
		if code == keyEnter {
			if len(value) > 0 {
				if err := a.input.Publish(ctx, scanner.Input{Value: string(value)}); err != nil {
					return err
				}
				value = value[:0]
			}
			continue
		}
		if digit, ok := keyCodeDigit(code); ok {
			value = append(value, digit)
		}
	}
}

type readResult struct {
	event []byte
	err   error
}

func (a *Adapter) readLoop(results chan<- readResult) {
	event := make([]byte, inputEventSize)
	for {
		_, err := io.ReadFull(a.device, event)
		copyOfEvent := append([]byte(nil), event...)
		results <- readResult{event: copyOfEvent, err: err}
		if err != nil {
			return
		}
	}
}

func (a *Adapter) Close() {
	a.once.Do(func() {
		a.input.Close()
		_ = a.device.Close()
	})
}

func keyCodeDigit(code uint16) (byte, bool) {
	if code >= 2 && code <= 11 {
		if code == 11 {
			return '0', true
		}
		return byte('1' + code - 2), true
	}
	return 0, false
}
