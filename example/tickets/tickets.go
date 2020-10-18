package tickets

import (
	"tickets/domain/tickettitle"
	"tickets/id"
)

type (
	TicketDescription    string
	TicketCommentMessage string
)

type (
	GetTicketByIDOut struct {
		ID          id.TicketID
		Title       tickettitle.TicketTitle
		Description TicketDescription
		Author      id.UserID
		Assignees   []id.UserID
	}
	CreateTicketIn struct {
		Title       tickettitle.TicketTitle
		Description TicketDescription
		Author      id.UserID
	}
	CreateTicketOut struct {
		ID          id.TicketID
		Title       tickettitle.TicketTitle
		Description TicketDescription
		Author      id.UserID
	}
	AssignUserToTicket struct {
		Ticket id.TicketID
		By     id.UserID
	}
	CloseTicketIn struct {
		Ticket id.TicketID
		By     id.UserID
	}
	CreateCommentIn struct {
		Ticket  id.TicketID
		Message TicketCommentMessage
		By      id.UserID
	}
	UnassigneUserFromTicketIn struct {
		User   id.UserID
		Ticket id.TicketID
		By     id.UserID
	}
	UpdateTicketIn struct {
		Ticket         id.TicketID
		NewDescription *TicketDescription
		NewTitle       *tickettitle.TicketTitle
		By             id.UserID
	}
)
