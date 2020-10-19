package tickets

import (
	"context"
	"fmt"
	"tickets/generated"
	"tickets/id"
	"tickets/service/tickets/io"
)

func (s *Service) GetTicketByID(
	ctx context.Context,
	tx generated.TransactionReader,
	in id.Ticket,
) (
	output io.GetTicketByIDOut,
	err error,
) {
	t, ok := tx.(transactionRead).store.state.tickets[in]
	if !ok {
		err = fmt.Errorf("not found")
		return
	}

	output.Assignees = make([]id.User, 0, len(t.Assignees))
	for u := range t.Assignees {
		output.Assignees = append(output.Assignees, u)
	}
	output.Author = t.Author
	output.Description = t.Description
	output.Title = t.Title
	output.ID = t.ID
	return
}
