package bridge

import (
	"context"
	"sync"

	"github.com/devicebridge/device-bridge/internal/message"
	"github.com/devicebridge/device-bridge/internal/source"
)

// Run starts the bridge runtime.
func (b *Bridge) Run(ctx context.Context) error {
	started := false
	b.runOnce.Do(func() {
		started = true
	})
	if !started {
		return ErrAlreadyRunning
	}

	runtimeCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		wg          sync.WaitGroup
		forwarderWG sync.WaitGroup
		firstErr    error
		errOnce     sync.Once
	)

	for _, name := range b.registry.Names() {
		src, err := b.registry.Create(name)
		if err != nil {
			errOnce.Do(func() {
				firstErr = err
				cancel()
			})
			continue
		}

		if src == nil {
			continue
		}

		wg.Add(1)
		go b.connectSource(runtimeCtx, src, &wg, &forwarderWG, &firstErr, &errOnce, cancel)
	}

	go func() {
		messages := b.bus.Subscribe()
		for {
			select {
			case msg := <-messages:
				_ = b.hub.Broadcast(msg)
			case <-b.bus.Done():
				for {
					select {
					case msg := <-messages:
						_ = b.hub.Broadcast(msg)
					default:
						return
					}
				}
			}
		}
	}()

	wg.Wait()
	forwarderWG.Wait()

	b.bus.Close()

	return firstErr
}

func (b *Bridge) connectSource(ctx context.Context, src source.Source, wg *sync.WaitGroup, forwarderWG *sync.WaitGroup, firstErr *error, errOnce *sync.Once, cancel context.CancelFunc) {
	defer wg.Done()

	out := make(chan message.Message, 100)

	sourceDone := make(chan error, 1)
	go func() {
		defer close(out)
		sourceDone <- src.Run(ctx, out)
	}()

	forwarderWG.Add(1)
	go func() {
		defer forwarderWG.Done()
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
