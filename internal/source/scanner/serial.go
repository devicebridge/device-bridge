package scanner

import (
	"bufio"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
)

// SerialAdapter converts a line-oriented serial stream into scanner inputs.
// The port is closed on cancellation so blocking reads can be interrupted.
type SerialAdapter struct {
	port  io.ReadCloser
	input *ChannelAdapter
	once  sync.Once
}

func NewSerialAdapter(port io.ReadCloser, buffer int) *SerialAdapter {
	if port == nil {
		panic("serial adapter port cannot be nil")
	}
	return &SerialAdapter{
		port:  port,
		input: NewChannelAdapter(buffer),
	}
}

func (a *SerialAdapter) Input() <-chan Input {
	return a.input.Input()
}

func (a *SerialAdapter) Run(ctx context.Context) error {
	readDone := make(chan readResult, 1)
	go a.readLoop(readDone)

	defer a.input.Close()
	for {
		select {
		case <-ctx.Done():
			a.Close()
			return ctx.Err()
		case result := <-readDone:
			if result.line != "" {
				if err := a.input.Publish(ctx, Input{Value: result.line}); err != nil {
					return err
				}
			}
			if errors.Is(result.err, io.EOF) {
				return nil
			}
			if result.err != nil {
				return result.err
			}
		}
	}
}

func (a *SerialAdapter) Close() {
	a.once.Do(func() {
		a.input.Close()
		_ = a.port.Close()
	})
}

type readResult struct {
	line string
	err  error
}

func (a *SerialAdapter) readLoop(results chan<- readResult) {
	reader := bufio.NewReader(a.port)
	for {
		line, err := reader.ReadString('\n')
		line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		results <- readResult{line: line, err: err}
		if err != nil {
			return
		}
	}
}
