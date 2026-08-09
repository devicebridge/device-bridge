package scanner

import (
	"context"
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

func (s *Scanner) Run(ctx context.Context, out chan<- message.Message) error {
	for {
		select {
		case in, ok := <-s.input:
			if !ok {
				return nil
			}

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

			select {
			case out <- msg:
			case <-ctx.Done():
				return ctx.Err()
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
