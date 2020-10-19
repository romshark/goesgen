package tickets

import "fmt"

type (
	TicketDescription    string
	TicketCommentMessage string
	TicketTitle          string
)

func ValidateTicketTitle(v TicketTitle) error {
	if v == "" {
		return fmt.Errorf("empty")
	}
	return nil
}

func ValidateCommentMessage(v TicketCommentMessage) error {
	if v == "" {
		return fmt.Errorf("empty")
	}
	return nil
}
