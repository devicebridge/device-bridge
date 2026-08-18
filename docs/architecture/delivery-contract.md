# Delivery and Backpressure Contract

## Delivery guarantee

Device Bridge provides **at-most-once delivery**. A message is not persisted and can be lost when:

- a source stops before publication;
- the runtime is shutting down;
- a WebSocket client disconnects;
- a client outbound queue is full.

Messages delivered to one client are written in enqueue order. There is no global ordering guarantee between multiple sources.

## Backpressure

- Source adapters use bounded input buffers. A blocking `Publish` must be supplied a cancellable context.
- The Bus uses a bounded buffer and `PublishCtx` returns when its context is cancelled or the Bus is closed.
- Each WebSocket client uses a bounded outbound queue of 16 messages.
- A full WebSocket queue is a client failure. The Hub removes and closes that client without stopping delivery to healthy clients.

The system does not retry messages and does not apply durable buffering. Callers that require stronger delivery guarantees must provide persistence and acknowledgements outside the current transport layer.
