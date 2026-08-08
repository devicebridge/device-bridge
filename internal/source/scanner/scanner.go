package scanner

import (
	"time"

	"github.com/devicebridge/device-bridge/internal/bus"
	"github.com/devicebridge/device-bridge/internal/message"
)

type Input struct {
	Value string
	Err   error
}

type Scanner struct {
	sourceID string
	input    <-chan Input
	bus      *bus.Bus
}

func New(sourceID string, input <-chan Input, bus *bus.Bus) *Scanner {
	return &Scanner{
		sourceID: sourceID,
		input:    input,
		bus:      bus,
	}
}

func (s *Scanner) Run(_ chan<- message.Message) error {
	for in := range s.input {
		if in.Err != nil {
			return in.Err
		}

		if in.Value == "" {
			continue
		}

		msg := message.Message{
			Source:    s.sourceID,
			Timestamp: time.Now().UnixMilli(),
			Payload:   in.Value,
		}

		s.bus.Publish(msg)
	}

	return nil
}
