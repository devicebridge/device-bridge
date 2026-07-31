package source

import (
	"testing"

	"github.com/devicebridge/device-bridge/internal/message"
)

type mockSource struct{}

func (mockSource) Run(chan<- message.Message) error {
	return nil
}

func TestSourceImplementsInterface(t *testing.T) {
	var _ Source = mockSource{}
}
