package auth

import (
	"context"
	"errors"
	"tickets/id"
)

type CtxKey int

const (
	CtxKeyUser CtxKey = 1
)

func User(ctx context.Context) (id.User, error) {
	u, ok := ctx.Value(CtxKeyUser).(id.User)
	if !ok {
		return "", ErrUnauthorized
	}
	return u, nil
}

var ErrUnauthorized = errors.New("unauthorized, access denied")
