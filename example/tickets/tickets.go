package tickets

import "fmt"

type (
	TicketDescription    string
	TicketCommentMessage string
	TicketTitle          string
	UserName             string
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

func ValidateUserName(v UserName) error {
	if len(v) < 4 {
		return fmt.Errorf("too short")
	}
	if len(v) > 64 {
		return fmt.Errorf("too long")
	}
	return nil
}
