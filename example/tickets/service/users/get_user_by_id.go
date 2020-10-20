package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"tickets/generated"
	"tickets/id"
	"tickets/service/users/io"
)

func (s *Service) GetUserByID(
	ctx context.Context,
	tx generated.TransactionReader,
	in id.User,
) (
	output io.GetUserByIDOut,
	err error,
) {
	row := tx.(transaction).tx.QueryRow(
		`SELECT name FROM users WHERE id = ?`,
		in,
	)
	switch err = row.Scan(&output.Name); err {
	case sql.ErrNoRows:
		err = errors.New("user not found")
		return
	case nil:
	default:
		err = fmt.Errorf("unexpected query error: %w", err)
		return
	}

	output.ID = in
	return
}
