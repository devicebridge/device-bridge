package bridge

// Run starts the bridge runtime.
func (b *Bridge) Run() {
	for msg := range b.bus.Subscribe() {
		_ = b.hub.Broadcast(msg)
	}
}
