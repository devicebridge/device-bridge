package bridge

import (
	"github.com/devicebridge/device-bridge/internal/message"
	"github.com/devicebridge/device-bridge/internal/source"
)

// Run starts the bridge runtime.
func (b *Bridge) Run() {
	for _, name := range b.registry.Names() {
		src, err := b.registry.Create(name)
		if err != nil {
			continue
		}

		go b.connectSource(src)
	}

	for msg := range b.bus.Subscribe() {
		_ = b.hub.Broadcast(msg)
	}
}

func (b *Bridge) connectSource(src source.Source) {
	out := make(chan message.Message, 100)

	go func() {
		defer close(out)
		_ = src.Run(out)
	}()

	for msg := range out {
		b.bus.Publish(msg)
	}
}
