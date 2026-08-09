package bridge

import (
	"context"
	"sync"

	"github.com/devicebridge/device-bridge/internal/message"
	"github.com/devicebridge/device-bridge/internal/source"
)

// Run starts the bridge runtime.
func (b *Bridge) Run(ctx context.Context) error {
	runtimeCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		wg       sync.WaitGroup
		firstErr error
		errOnce  sync.Once
	)

	for _, name := range b.registry.Names() {
		src, err := b.registry.Create(name)
		if err != nil {
			continue
		}

		wg.Add(1)
		go b.connectSource(runtimeCtx, src, &wg, &firstErr, &errOnce, cancel)
	}

	go func() {
		for msg := range b.bus.Subscribe() {
			_ = b.hub.Broadcast(msg)
		}
	}()

	wg.Wait()

	b.bus.Close()

	return firstErr
}

func (b *Bridge) connectSource(ctx context.Context, src source.Source, wg *sync.WaitGroup, firstErr *error, errOnce *sync.Once, cancel context.CancelFunc) {
	defer wg.Done()

	out := make(chan message.Message, 100)

	sourceDone := make(chan error, 1)
	go func() {
		defer close(out)
		sourceDone <- src.Run(ctx, out)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for msg := range out {
			if err := b.bus.PublishCtx(ctx, msg); err != nil {
				return
			}
		}
	}()

	err := <-sourceDone

	if err != nil && err != context.Canceled {
		errOnce.Do(func() {
			*firstErr = err
			cancel()
		})
	}
}
