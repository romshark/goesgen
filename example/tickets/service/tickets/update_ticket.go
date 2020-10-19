package tickets

import (
	"context"
	"fmt"
	"tickets"
	"tickets/auth"
	"tickets/generated"
	"tickets/service/tickets/io"
)

func (s *Service) UpdateTicket(
	ctx context.Context,
	tx generated.TransactionReader,
	in io.UpdateTicketIn,
) (events []generated.Event, err error) {
	client, err := auth.User(ctx)
	if err != nil {
		return nil, err
	}

	t, ok := tx.(transaction).store.state.tickets[in.Ticket]
	if !ok {
		return nil, fmt.Errorf("ticket %s not found", in.Ticket)
	}

	if in.NewDescription != nil && *in.NewDescription != t.Description {
		events = append(events, generated.EventTicketDescriptionChanged{
			Ticket:         t.ID,
			NewDescription: *in.NewDescription,
			By:             client,
		})
	}

	if in.NewTitle != nil {
		if err := tickets.ValidateTicketTitle(*in.NewTitle); err != nil {
			return nil, fmt.Errorf("invalid new title: %w", err)
		}
		if t.Title != *in.NewTitle {
			events = append(events, generated.EventTicketTitleChanged{
				Ticket:   t.ID,
				NewTitle: *in.NewTitle,
				By:       client,
			})
		}
	}

	return events, nil
}
