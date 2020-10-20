package tickets

import (
	"context"
	"log"
	"sync"
	"tickets"
	"tickets/generated"
	"tickets/id"
	"time"
)

type user struct {
	ID         id.User
	AssignedTo map[*ticket]struct{}
	AuthorOf   map[*ticket]struct{}
}

type ticketComment struct {
	Ticket  *ticket
	Message tickets.TicketCommentMessage
	Author  *user
}

type ticket struct {
	State       generated.ProjectionTicketState
	ID          id.Ticket
	Title       tickets.TicketTitle
	Description tickets.TicketDescription
	Author      *user
	Comments    []ticketComment
	Assignees   map[*user]struct{}
}

type StoreState struct {
	tickets map[id.Ticket]*ticket
	users   map[id.User]*user
}

func NewStoreState() *StoreState {
	return &StoreState{
		tickets: make(map[id.Ticket]*ticket),
		users:   make(map[id.User]*user),
	}
}

// Clone returns a deep copy of the store state
func (s *StoreState) Clone() *StoreState {
	t := make(map[id.Ticket]*ticket, len(s.tickets))
	for k, v := range s.tickets {
		c := *v
		c.Comments = make([]ticketComment, len(c.Comments))
		for i, v := range v.Comments {
			c.Comments[i] = ticketComment{
				Ticket:  &c,
				Message: v.Message,
			}
		}
		t[k] = &c
	}

	u := make(map[id.User]*user, len(s.users))
	for k, v := range s.users {
		userCopy := &user{
			ID:         v.ID,
			AssignedTo: make(map[*ticket]struct{}, len(v.AssignedTo)),
			AuthorOf:   make(map[*ticket]struct{}, len(v.AuthorOf)),
		}

		// Re-link tickets assigned to
		for k := range v.AssignedTo {
			userCopy.AssignedTo[t[k.ID]] = struct{}{}
		}

		// Re-link tickets author of
		for k := range v.AuthorOf {
			userCopy.AuthorOf[t[k.ID]] = struct{}{}
		}

		u[k] = userCopy
	}

	for _, v := range t {
		// Re-link assignees
		a := make(map[*user]struct{}, len(v.Assignees))
		for k := range v.Assignees {
			a[u[k.ID]] = struct{}{}
		}
		v.Assignees = a

		// Re-link comment authors
		for _, v := range v.Comments {
			v.Author = u[v.Author.ID]
		}
	}

	return &StoreState{
		tickets: t,
		users:   u,
	}
}

type Store struct {
	state             *StoreState
	projectionVersion generated.EventlogVersion
	lock              sync.RWMutex
}

func NewStore() *Store {
	return &Store{
		state:             NewStoreState(),
		projectionVersion: "0",
	}
}

type transaction struct {
	store           *Store
	previousState   *StoreState
	previousVersion generated.EventlogVersion
}

func (t transaction) Commit() { t.store.lock.Unlock() }

func (t transaction) Rollback() {
	t.store.state = t.previousState
	t.store.projectionVersion = t.previousVersion
	t.store.lock.Unlock()
}

type transactionRead struct{ store *Store }

func (t transactionRead) Complete() { t.store.lock.RUnlock() }

func (s *Store) NewTransactionReadWriter() generated.StoreTransactionReadWriter {
	s.lock.Lock()
	return transaction{
		store:           s,
		previousState:   s.state.Clone(),
		previousVersion: s.projectionVersion,
	}
}

func (s *Store) NewTransactionReader() generated.StoreTransactionReader {
	s.lock.RLock()
	return transactionRead{s}
}

// ProjectionVersion returns the current projection version.
// Returns an empty string if the projection wasn't initialized yet.
// In case an empty string is returned the service will fallback
// to the begin offset version of the eventlog.
func (s *Store) ProjectionVersion(
	context.Context,
	generated.TransactionReader,
) (generated.EventlogVersion, error) {
	return s.projectionVersion, nil
}

func (s *Store) UpdateProjectionVersion(
	ctx context.Context,
	tx generated.TransactionWriter,
	v generated.EventlogVersion,
) error {
	s.projectionVersion = v
	return nil
}

func (s *Store) ApplyEventTicketClosed(
	ctx context.Context,
	tx generated.TransactionWriter,
	v generated.EventlogVersion,
	tm time.Time,
	e generated.EventTicketClosed,
) error {
	log.Printf("ApplyEventTicketClosed: (%s) %#v", tm, e)
	s.state.tickets[e.Ticket].State = generated.ProjectionTicketStateClosed
	return nil
}

func (s *Store) ApplyEventTicketCommented(
	ctx context.Context,
	tx generated.TransactionWriter,
	v generated.EventlogVersion,
	tm time.Time,
	e generated.EventTicketCommented,
) error {
	log.Printf("ApplyEventTicketCommented: (%s) %#v", tm, e)
	t := s.state.tickets[e.Ticket]
	t.Comments = append(t.Comments, ticketComment{
		Ticket:  t,
		Message: e.Message,
		Author:  s.state.users[e.By],
	})
	return nil
}

func (s *Store) ApplyEventTicketCreated(
	ctx context.Context,
	tx generated.TransactionWriter,
	v generated.EventlogVersion,
	tm time.Time,
	e generated.EventTicketCreated,
) error {
	log.Printf("ApplyEventTicketCreated: (%s) %#v", tm, e)
	s.state.tickets[e.Id] = &ticket{
		State:       generated.ProjectionTicketStateNew,
		ID:          e.Id,
		Title:       e.Title,
		Description: e.Description,
		Author:      s.state.users[e.Author],
		Comments:    nil,
		Assignees:   map[*user]struct{}{},
	}
	return nil
}

func (s *Store) ApplyEventTicketDescriptionChanged(
	ctx context.Context,
	tx generated.TransactionWriter,
	v generated.EventlogVersion,
	tm time.Time,
	e generated.EventTicketDescriptionChanged,
) error {
	log.Printf("ApplyEventTicketDescriptionChanged: (%s) %#v", tm, e)
	s.state.tickets[e.Ticket].Description = e.NewDescription
	return nil
}

func (s *Store) ApplyEventTicketTitleChanged(
	ctx context.Context,
	tx generated.TransactionWriter,
	v generated.EventlogVersion,
	tm time.Time,
	e generated.EventTicketTitleChanged,
) error {
	log.Printf("ApplyEventTicketTitleChanged: (%s) %#v", tm, e)
	s.state.tickets[e.Ticket].Title = e.NewTitle
	return nil
}

func (s *Store) ApplyEventUserAssignedToTicket(
	ctx context.Context,
	tx generated.TransactionWriter,
	v generated.EventlogVersion,
	tm time.Time,
	e generated.EventUserAssignedToTicket,
) error {
	log.Printf("ApplyEventUserAssignedToTicket: (%s) %#v", tm, e)
	u := s.state.users[e.User]
	t := s.state.tickets[e.Ticket]
	t.Assignees[u] = struct{}{}
	return nil
}

func (s *Store) ApplyEventUserUnassignedFromTicket(
	ctx context.Context,
	tx generated.TransactionWriter,
	v generated.EventlogVersion,
	tm time.Time,
	e generated.EventUserUnassignedFromTicket,
) error {
	log.Printf("ApplyEventUserUnassignedFromTicket: (%s) %#v", tm, e)
	delete(s.state.tickets[e.Ticket].Assignees, s.state.users[e.User])
	return nil
}

func (s *Store) ApplyEventUserCreated(
	ctx context.Context,
	tx generated.TransactionWriter,
	v generated.EventlogVersion,
	tm time.Time,
	e generated.EventUserCreated,
) error {
	log.Printf("ApplyEventUserCreated: (%s) %#v", tm, e)
	s.state.users[e.Id] = &user{
		ID:         e.Id,
		AssignedTo: map[*ticket]struct{}{},
		AuthorOf:   map[*ticket]struct{}{},
	}
	return nil
}
