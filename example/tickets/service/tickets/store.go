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

type ticketComment struct {
	Ticket  *ticket
	Message tickets.TicketCommentMessage
	Author  id.User
}

type ticket struct {
	State       generated.ProjectionTicketState
	ID          id.Ticket
	Title       tickets.TicketTitle
	Description tickets.TicketDescription
	Author      id.User
	Comments    []ticketComment
	Assignees   map[id.User]struct{}
}

func (t *ticket) Clone() *ticket {
	c := *t
	c.Comments = make([]ticketComment, len(c.Comments))
	for i, v := range t.Comments {
		c.Comments[i] = ticketComment{
			Ticket:  &c,
			Message: v.Message,
			Author:  v.Author,
		}
	}
	c.Assignees = make(map[id.User]struct{}, len(t.Assignees))
	for k := range t.Assignees {
		c.Assignees[k] = struct{}{}
	}
	return &c
}

type StoreState struct {
	tickets map[id.Ticket]*ticket
}

func NewStoreState() *StoreState {
	return &StoreState{
		tickets: make(map[id.Ticket]*ticket),
	}
}

// Clone returns a deep copy of the store state
func (s *StoreState) Clone() *StoreState {
	t := make(map[id.Ticket]*ticket, len(s.tickets))
	for k, v := range s.tickets {
		t[k] = v.Clone()
	}
	return &StoreState{
		tickets: t,
	}
}

type Store struct {
	state             *StoreState
	projectionVersion generated.EventlogVersion
	lock              sync.RWMutex
}

func NewStore() *Store {
	return &Store{
		state: NewStoreState(),
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

// ApplyEventTicketClosed applies event TicketClosed to the projection.
// The projection must update its local projection version
// to the one that is provided.
func (s *Store) ApplyEventTicketClosed(
	ctx context.Context,
	tx generated.TransactionWriter,
	v generated.EventlogVersion,
	tm time.Time,
	e generated.EventTicketClosed,
) error {
	log.Printf("ApplyEventTicketClosed: (%s) %#v", tm, e)
	s.state.tickets[e.Ticket].State = generated.ProjectionTicketStateClosed
	s.projectionVersion = v
	return nil
}

// ApplyEventTicketCommented applies event TicketCommented to the projection.
// The projection must update its local projection version
// to the one that is provided.
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
		Author:  e.By,
	})
	s.projectionVersion = v
	return nil
}

// ApplyEventTicketCreated applies event TicketCreated to the projection.
// The projection must update its local projection version
// to the one that is provided.
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
		Author:      e.Author,
		Comments:    nil,
		Assignees:   map[id.User]struct{}{},
	}
	s.projectionVersion = v
	return nil
}

// ApplyEventTicketDescriptionChanged applies event TicketDescriptionChanged to the projection.
// The projection must update its local projection version
// to the one that is provided.
func (s *Store) ApplyEventTicketDescriptionChanged(
	ctx context.Context,
	tx generated.TransactionWriter,
	v generated.EventlogVersion,
	tm time.Time,
	e generated.EventTicketDescriptionChanged,
) error {
	log.Printf("ApplyEventTicketDescriptionChanged: (%s) %#v", tm, e)
	s.state.tickets[e.Ticket].Description = e.NewDescription
	s.projectionVersion = v
	return nil
}

// ApplyEventTicketTitleChanged applies event TicketTitleChanged to the projection.
// The projection must update its local projection version
// to the one that is provided.
func (s *Store) ApplyEventTicketTitleChanged(
	ctx context.Context,
	tx generated.TransactionWriter,
	v generated.EventlogVersion,
	tm time.Time,
	e generated.EventTicketTitleChanged,
) error {
	log.Printf("ApplyEventTicketTitleChanged: (%s) %#v", tm, e)
	s.state.tickets[e.Ticket].Title = e.NewTitle
	s.projectionVersion = v
	return nil
}

// ApplyEventUserAssignedToTicket applies event UserAssignedToTicket to the projection.
// The projection must update its local projection version
// to the one that is provided.
func (s *Store) ApplyEventUserAssignedToTicket(
	ctx context.Context,
	tx generated.TransactionWriter,
	v generated.EventlogVersion,
	tm time.Time,
	e generated.EventUserAssignedToTicket,
) error {
	log.Printf("ApplyEventUserAssignedToTicket: (%s) %#v", tm, e)
	s.state.tickets[e.Ticket].Assignees[e.User] = struct{}{}
	s.projectionVersion = v
	return nil
}

// ApplyEventUserUnassignedFromTicket applies event UserUnassignedFromTicket to the projection.
// The projection must update its local projection version
// to the one that is provided.
func (s *Store) ApplyEventUserUnassignedFromTicket(
	ctx context.Context,
	tx generated.TransactionWriter,
	v generated.EventlogVersion,
	tm time.Time,
	e generated.EventUserUnassignedFromTicket,
) error {
	log.Printf("ApplyEventUserUnassignedFromTicket: (%s) %#v", tm, e)
	delete(s.state.tickets[e.Ticket].Assignees, e.User)
	s.projectionVersion = v
	return nil
}
