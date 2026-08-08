package scanner

import (
	"time"

	"github.com/devicebridge/device-bridge/internal/message"
)

type Input struct {
	Value string
	Err   error
}

type Scanner struct {
	sourceID string
	input    <-chan Input
}

func New(sourceID string, input <-chan Input) *Scanner {
	return &Scanner{
		sourceID: sourceID,
		input:    input,
	}
}

func (s *Scanner) Run(out chan<- message.Message) error {
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

		out <- msg
	}

	return nil
}
