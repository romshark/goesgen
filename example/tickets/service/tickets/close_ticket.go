package tickets

import (
	"context"
	"fmt"
	"tickets/auth"
	"tickets/generated"
	"tickets/service/tickets/io"
)

func (s *Service) CloseTicket(
	ctx context.Context,
	tx generated.TransactionReader,
	in io.CloseTicketIn,
) ([]generated.Event, error) {
	client, err := auth.User(ctx)
	if err != nil {
		return nil, err
	}

	if _, ok := tx.(transaction).store.state.users[client]; !ok {
		return nil, fmt.Errorf("user %s not found", client)
	}

	t, ok := tx.(transaction).store.state.tickets[in.Ticket]
	if !ok {
		return nil, fmt.Errorf("ticket %s not found", in.Ticket)
	}
	if t.State == generated.ProjectionTicketStateClosed {
		return nil, fmt.Errorf("ticket already closed")
	}

	return []generated.Event{
		generated.EventTicketClosed{
			Ticket: in.Ticket,
			By:     client,
		},
	}, nil
}
