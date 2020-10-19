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

func (s *Service) CreateTicket(
	ctx context.Context,
	tx generated.TransactionReader,
	in io.CreateTicketIn,
) (
	output io.CreateTicketOut,
	events []generated.Event,
	err error,
) {
	client, err := auth.User(ctx)
	if err != nil {
		return
	}

	if err = tickets.ValidateTicketTitle(in.Title); err != nil {
		err = fmt.Errorf("invalid ticket title: %w", err)
		return
	}

	newID := id.Ticket(id.New())

	output = io.CreateTicketOut{
		Author:      client,
		Description: in.Description,
		Title:       in.Title,
		ID:          newID,
	}
	events = []generated.Event{
		generated.EventTicketCreated{
			Id:          newID,
			Title:       in.Title,
			Description: in.Description,
			Author:      client,
		},
	}
	return
}
