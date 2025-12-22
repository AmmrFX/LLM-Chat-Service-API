package service

// ChatService defines the interface for chat operations
type ChatService interface {
	ProcessChat(req *ChatRequest) (string, error)
	ProcessChatStream(req *ChatRequest, onToken func(string) error) (string, error)
}
