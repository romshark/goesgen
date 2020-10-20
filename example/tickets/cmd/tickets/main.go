package main

import (
	"context"
	"log"
	"os"
	"tickets/auth"
	"tickets/generated"
	"tickets/service"
	stickets "tickets/service/tickets"
	"tickets/service/tickets/io"
	susers "tickets/service/users"
	usersio "tickets/service/users/io"

	"github.com/romshark/eventlog/client"
	"github.com/romshark/eventlog/eventlog"
	"github.com/romshark/eventlog/eventlog/inmem"
)

func main() {
	lErr := log.New(os.Stderr, "ERR", log.LstdFlags)
	e := client.NewInmem(eventlog.New(inmem.New()))
	c := client.New(e)

	// Initialize in-memory tickets service
	serviceTickets := generated.NewServiceTickets(
		stickets.New(),
		stickets.NewStore(),
		&service.EventlogAdapter{Client: c},
		lErr,
		generated.ServiceOptions{},
	)

	// Initialize in-memory SQL-based users service
	usersStore, err := susers.NewInmemSQLStore()
	if err != nil {
		log.Fatalf("initializing users service store: %s", err)
	}
	serviceUsers := generated.NewServiceUsers(
		susers.New(),
		usersStore,
		&service.EventlogAdapter{Client: c},
		lErr,
		generated.ServiceOptions{},
	)

	// Create user A
	userA, _, _, err := serviceUsers.CreateUser(
		context.Background(),
		usersio.CreateUserIn{
			Name: "User A",
		},
	)
	if err != nil {
		log.Fatalf("creating user A: %s", err)
	}
	log.Printf("Created user %q: %s", userA.Name, userA.ID)

	// Create user B
	userB, _, _, err := serviceUsers.CreateUser(
		context.Background(),
		usersio.CreateUserIn{
			Name: "User B",
		},
	)
	if err != nil {
		log.Fatalf("creating user B: %s", err)
	}
	log.Printf("Created user %q: %s", userB.Name, userB.ID)

	ctxAsUserA := context.WithValue(
		context.Background(),
		auth.CtxKeyUser,
		userA.ID,
	)

	if _, err := serviceTickets.Sync(context.Background(), nil); err != nil {
		log.Fatalf("Synchronizing tickets service: %s", err)
	}

	// Create a ticket
	newTicket, _, tm, err := serviceTickets.CreateTicket(
		ctxAsUserA,
		io.CreateTicketIn{
			Title:       "Example Ticket",
			Description: "This is an example ticket",
		},
	)
	if err != nil {
		log.Fatalf("creating new ticket: %s", err)
	}
	log.Printf("Ticket created (%s): %#v", tm, newTicket)

	{ // Assign user B to ticket
		_, tm, err = serviceTickets.AssignUserToTicket(
			ctxAsUserA,
			io.AssignUserToTicketIn{
				User:   userB.ID,
				Ticket: newTicket.ID,
			},
		)
		if err != nil {
			log.Fatalf("assigning user %s to ticket: %s", userB, err)
		}
		log.Printf("User B assigned to ticket")
	}

	{
		v, err := serviceTickets.ProjectionVersion(context.Background())
		if err != nil {
			log.Fatalf("Reading before version: %s", err)
		}
		log.Printf("Projection version before sync: %s", v)
		after, err := serviceTickets.Sync(context.Background(), nil)
		if err != nil {
			log.Fatalf("Synchronizing: %s", err)
		}
		log.Printf("Projection version after sync: %s", after)
	}

	{ // Get ticket
		foundTicket, err := serviceTickets.GetTicketByID(
			ctxAsUserA,
			newTicket.ID,
		)
		if err != nil {
			log.Fatalf("getting ticket by ID: %s", err)
		}
		log.Printf("Found ticket: %#v", foundTicket)
	}
}
