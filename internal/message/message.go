package message

// Message represents a transport message exchanged between
// sources and connected clients.
type Message struct {
	Source    string `json:"source"`
	Timestamp int64  `json:"timestamp"`
	Payload   string `json:"payload"`
}
