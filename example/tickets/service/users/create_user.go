package users

import (
	"context"
	"database/sql"
	"fmt"
	"tickets"
	"tickets/generated"
	"tickets/id"
	"tickets/service/users/io"
)

func (s *Service) CreateUser(
	ctx context.Context,
	tx generated.TransactionReader,
	in io.CreateUserIn,
) (
	output io.CreateUserOut,
	events []generated.Event,
	err error,
) {
	if err = tickets.ValidateUserName(in.Name); err != nil {
		err = fmt.Errorf("invalid username: %w", err)
		return
	}

	// Make sure the user name isn't yet reserved by an existing user
	row := tx.(transaction).tx.QueryRow(
		`SELECT id FROM users WHERE name = ?`,
		in.Name,
	)
	var uid id.User
	if err = row.Scan(&uid); err != sql.ErrNoRows {
		if err == nil {
			err = fmt.Errorf("username reserved")
			return
		}
		err = fmt.Errorf("unexpected query error: %w", err)
		return
	} else {
		err = nil
	}

	output.ID = id.User(id.New())
	output.Name = in.Name
	events = []generated.Event{
		generated.EventUserCreated{
			Id:   output.ID,
			Name: in.Name,
		},
	}
	return
}
