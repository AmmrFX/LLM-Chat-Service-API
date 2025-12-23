package service

import (
	"fmt"

	apperror "llm-chat-service/internal/error"
)

// ------------------------------------------------------------------------------------------------------
func (r *ChatRequest) Validate() error {
	if len(r.Messages) == 0 {
		return apperror.NewValidationError("messages cannot be empty", nil)
	}

	// Validate each message
	for i, msg := range r.Messages {
		if msg.Role != "user" && msg.Role != "assistant" {
			return apperror.NewValidationError(
				fmt.Sprintf("invalid role '%s' at index %d: must be 'user' or 'assistant'", msg.Role, i),
				nil,
			)
		}
		if msg.Content == "" {
			return apperror.NewValidationError(
				fmt.Sprintf("empty content at index %d", i),
				nil,
			)
		}
	}

	// Last message must be from user
	lastMsg := r.Messages[len(r.Messages)-1]
	if lastMsg.Role != "user" {
		return apperror.NewValidationError(
			fmt.Sprintf("last message must be from user, got '%s'", lastMsg.Role),
			nil,
		)
	}

	return nil
}
