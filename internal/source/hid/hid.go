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
	keyLeftShift   = 42
	keyRightShift  = 54
	keyCapsLock    = 58
	keyBackspace   = 14
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
	state := keyboardState{}
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
		if typeID != evKey {
			continue
		}
		if code == keyLeftShift || code == keyRightShift {
			state.shift = keyValue != keyRelease
			continue
		}
		if code == keyCapsLock && keyValue == keyRelease {
			state.caps = !state.caps
			continue
		}
		if keyValue != keyRelease {
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
		if code == keyBackspace {
			if len(value) > 0 {
				value = value[:len(value)-1]
			}
			continue
		}
		if character, ok := keyCodeCharacter(code, state); ok {
			value = append(value, character)
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

type keyboardState struct {
	shift bool
	caps  bool
}

func keyCodeCharacter(code uint16, state keyboardState) (byte, bool) {
	if code >= 2 && code <= 11 {
		if code == 11 {
			return shifted('0', ')', state.shift)
		}
		plain := byte('1' + code - 2)
		shiftedValue := byte("!@#$%^&*("[code-2])
		return shifted(plain, shiftedValue, state.shift)
	}
	letters := map[uint16]byte{
		16: 'q', 17: 'w', 18: 'e', 19: 'r', 20: 't', 21: 'y', 22: 'u', 23: 'i', 24: 'o', 25: 'p',
		30: 'a', 31: 's', 32: 'd', 33: 'f', 34: 'g', 35: 'h', 36: 'j', 37: 'k', 38: 'l',
		44: 'z', 45: 'x', 46: 'c', 47: 'v', 48: 'b', 49: 'n', 50: 'm',
	}
	if plain, ok := letters[code]; ok {
		if state.shift != state.caps {
			plain -= 'a' - 'A'
		}
		return plain, true
	}
	characters := map[uint16][2]byte{
		12: {'-', '_'}, 13: {'=', '+'}, 26: {'[', '{'}, 27: {']', '}'},
		39: {';', ':'}, 40: {'\'', '"'}, 41: {'`', '~'}, 43: {'\\', '|'},
		51: {',', '<'}, 52: {'.', '>'}, 53: {'/', '?'}, 57: {' ', ' '},
		71: {'7', '7'}, 72: {'8', '8'}, 73: {'9', '9'}, 75: {'4', '4'}, 76: {'5', '5'},
		77: {'6', '6'}, 79: {'1', '1'}, 80: {'2', '2'}, 81: {'3', '3'}, 82: {'0', '0'},
		83: {'.', '.'},
	}
	values, ok := characters[code]
	if !ok {
		return 0, false
	}
	return shifted(values[0], values[1], state.shift)
}

func shifted(plain, withShift byte, shift bool) (byte, bool) {
	if shift {
		return withShift, true
	}
	return plain, true
}
