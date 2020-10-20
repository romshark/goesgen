package tickets

import (
	"context"
	"fmt"
	"tickets"
	"tickets/auth"
	"tickets/generated"
	"tickets/id"
	"tickets/service/tickets/io"
)

func (s *Service) CreateComment(
	ctx context.Context,
	tx generated.TransactionReader,
	in io.CreateCommentIn,
) (
	output io.CreateCommentOut,
	events []generated.Event,
	err error,
) {
	client, err := auth.User(ctx)
	if err != nil {
		return
	}

	if _, ok := tx.(transaction).store.state.users[client]; !ok {
		err = fmt.Errorf("user %s not found", client)
		return
	}

	if _, ok := tx.(transaction).store.state.tickets[in.Ticket]; !ok {
		err = fmt.Errorf("ticket %s not found", in.Ticket)
		return
	}

	if err = tickets.ValidateCommentMessage(in.Message); err != nil {
		err = fmt.Errorf("invalid comment message: %w", err)
		return
	}

	newID := id.Comment(id.New())

	events = []generated.Event{
		generated.EventTicketCommented{
			Id:      newID,
			Ticket:  in.Ticket,
			Message: in.Message,
			By:      client,
		},
	}
	output = io.CreateCommentOut{
		Id:      newID,
		Ticket:  in.Ticket,
		Message: in.Message,
		Author:  client,
	}
	return
}
