package source

import (
	"context"

	"github.com/devicebridge/device-bridge/internal/message"
)

// Source produces transport messages.
type Source interface {
	Run(ctx context.Context, out chan<- message.Message) error
}
