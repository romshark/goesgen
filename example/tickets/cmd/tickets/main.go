package main

import (
	"context"
	"log"
	"os"
	"tickets/auth"
	"tickets/generated"
	"tickets/id"
	service "tickets/service/tickets"
	"tickets/service/tickets/io"

	"github.com/romshark/eventlog/client"
	"github.com/romshark/eventlog/eventlog"
	"github.com/romshark/eventlog/eventlog/inmem"
)

func main() {
	lErr := log.New(os.Stderr, "ERR", log.LstdFlags)
	e := client.NewInmem(eventlog.New(inmem.New()))
	c := client.New(e)

	s := generated.NewServiceTickets(
		service.New(),
		service.NewStore(),
		&EventlogAdapter{c},
		lErr,
	)

	ctxAsUserA := context.WithValue(
		context.Background(),
		auth.CtxKeyUser,
		id.User("UserA"),
	)

	// Create a ticket
	newTicket, _, tm, err := s.CreateTicket(
		ctxAsUserA,
		io.CreateTicketIn{
			Title:       "Example Ticket",
			Description: "This is an example ticket",
			Author:      "UserA",
		},
	)
	if err != nil {
		log.Fatalf("creating new ticket: %s", err)
	}
	log.Printf("Ticket created (%s): %#v", tm, newTicket)

	// Assign user B to ticket
	_, tm, err = s.AssignUserToTicket(
		ctxAsUserA,
		io.AssignUserToTicketIn{
			User:   "UserB",
			Ticket: newTicket.ID,
		},
	)
	if err != nil {
		log.Fatalf("assigning user to ticket: %s", err)
	}
	log.Printf("User B assigned to ticket")

	v, err := s.ProjectionVersion(context.Background())
	if err != nil {
		log.Fatalf("Reading before version: %s", err)
	}
	log.Printf("Projection version before sync: %s", v)
	after, err := s.Sync(
		context.Background(),
		nil,
	)
	if err != nil {
		log.Fatalf("Synchronizing: %s", err)
	}
	log.Printf("Projection version after sync: %s", after)

	// Get ticket
	foundTicket, err := s.GetTicketByID(
		ctxAsUserA,
		newTicket.ID,
	)
	if err != nil {
		log.Fatalf("getting ticket by ID: %s", err)
	}
	log.Printf("Found ticket: %#v", foundTicket)
}
