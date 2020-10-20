package io

import (
	"tickets"
	"tickets/id"
)

type (
	GetTicketByIDOut struct {
		ID          id.Ticket
		Title       tickets.TicketTitle
		Description tickets.TicketDescription
		Author      id.User
		Assignees   []id.User
	}
	CreateTicketIn struct {
		Title       tickets.TicketTitle
		Description tickets.TicketDescription
	}
	CreateTicketOut struct {
		ID          id.Ticket
		Title       tickets.TicketTitle
		Description tickets.TicketDescription
		Author      id.User
	}
	CloseTicketIn struct {
		Ticket id.Ticket
	}
	CreateCommentIn struct {
		Ticket  id.Ticket
		Message tickets.TicketCommentMessage
	}
	CreateCommentOut struct {
		Id      id.Comment
		Ticket  id.Ticket
		Message tickets.TicketCommentMessage
		Author  id.User
	}
	AssignUserToTicketIn struct {
		User   id.User
		Ticket id.Ticket
	}
	UnassignUserFromTicketIn struct {
		User   id.User
		Ticket id.Ticket
	}
	UpdateTicketIn struct {
		Ticket         id.Ticket
		NewDescription *tickets.TicketDescription
		NewTitle       *tickets.TicketTitle
	}
)
