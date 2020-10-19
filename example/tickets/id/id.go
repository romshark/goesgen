package id

import uuid "github.com/satori/go.uuid"

type (
	User    string
	Ticket  string
	Comment string
)

func New() string {
	u1 := uuid.NewV4()
	return u1.String()
}
