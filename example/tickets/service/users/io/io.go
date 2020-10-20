package io

import (
	"tickets"
	"tickets/id"
)

type (
	GetUserByIDOut struct {
		ID   id.User
		Name tickets.UserName
	}
	CreateUserIn struct {
		Name tickets.UserName
	}
	CreateUserOut struct {
		ID   id.User
		Name tickets.UserName
	}
)
