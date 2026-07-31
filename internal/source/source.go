package source

import "github.com/devicebridge/device-bridge/internal/message"

// Source produces transport messages.
type Source interface {
	Run(chan<- message.Message) error
}
