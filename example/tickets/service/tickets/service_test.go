package tickets_test

import (
	"context"
	"log"
	"os"
	"testing"
	"tickets"
	"tickets/auth"
	"tickets/generated"
	"tickets/id"
	"tickets/service"
	stickets "tickets/service/tickets"
	"tickets/service/tickets/io"
	"time"

	"github.com/romshark/eventlog/client"
	"github.com/romshark/eventlog/eventlog"
	"github.com/romshark/eventlog/eventlog/inmem"
	"github.com/stretchr/testify/require"
)

func TestCreateTicket(t *testing.T) {
	s := NewSetup(t, generated.EventUserCreated{
		Id:   "user_foo",
		Name: "Foo",
	})

	o, e, tm, err := s.Service.CreateTicket(
		context.WithValue(
			context.Background(),
			auth.CtxKeyUser,
			id.User("user_foo"),
		),
		io.CreateTicketIn{
			Description: "test description",
			Title:       "test title",
		},
	)

	// Check output
	require.NoError(t, err)
	require.WithinDuration(t, time.Now(), tm, time.Second)
	require.Len(t, e, 1)
	require.IsType(t, generated.EventTicketCreated{}, e[0])

	require.Equal(t, id.User("user_foo"), o.Author)
	require.Equal(t,
		tickets.TicketDescription("test description"),
		o.Description,
	)
	require.Equal(t,
		tickets.TicketTitle("test title"),
		o.Title,
	)
	require.Len(t, o.ID, 36)

	// Check pushed events
	require.Equal(t, uint64(2), s.Eventlog.Version())
	s.checkEvent(1,
		func(tm time.Time, e generated.Event) {
			require.IsType(t, generated.EventTicketCreated{}, e)
			v := e.(generated.EventTicketCreated)
			require.Equal(t, o.ID, v.Id)
			require.Equal(t, id.User("user_foo"), v.Author)
			require.Equal(t,
				tickets.TicketDescription("test description"),
				v.Description,
			)
			require.Equal(t,
				tickets.TicketTitle("test title"),
				v.Title,
			)
		},
	)
}

func TestCreateTicketErrInvalidTitle(t *testing.T) {
	s := NewSetup(t, generated.EventUserCreated{
		Id:   "user_foo",
		Name: "Foo",
	})

	o, e, tm, err := s.Service.CreateTicket(
		context.WithValue(
			context.Background(),
			auth.CtxKeyUser,
			id.User("user_foo"),
		),
		io.CreateTicketIn{
			Description: "test description",
			Title:       "", // Illegal
		},
	)

	// Check output
	require.Error(t, err)
	require.Equal(t, "invalid ticket title: empty", err.Error())
	require.Zero(t, tm)
	require.Zero(t, e)
	require.Zero(t, o)

	// Check pushed events
	require.Equal(t, uint64(1), s.Eventlog.Version())
}

type Setup struct {
	t        *testing.T
	Service  *generated.ServiceTickets
	Eventlog *eventlog.EventLog
}

func NewSetup(t *testing.T, events ...generated.Event) Setup {
	lErr := log.New(os.Stderr, "ERR", log.LstdFlags)
	l := eventlog.New(inmem.New())
	c := client.New(client.NewInmem(l))
	srv := generated.NewServiceTickets(
		stickets.New(),
		stickets.NewStore(),
		&service.EventlogAdapter{Client: c},
		lErr,
		generated.ServiceOptions{},
	)

	s := Setup{
		t:        t,
		Eventlog: l,
		Service:  srv,
	}

	s.appendEvents(events...)

	_, err := srv.Sync(context.Background(), nil)
	require.NoError(t, err)

	return s
}

func (s Setup) appendEvents(e ...generated.Event) {
	payloads := make([][]byte, len(e))
	for i, e := range e {
		b, err := generated.EncodeEventJSON(e)
		require.NoError(s.t, err)
		payloads[i] = b
	}
	_, _, _, err := s.Eventlog.AppendMulti(payloads...)
	require.NoError(s.t, err)
}

func (s Setup) checkEvent(
	offset uint64,
	onEvent ...func(tm time.Time, e generated.Event),
) {
	for i, check := range onEvent {
		_, err := s.Eventlog.Scan(
			offset+uint64(i),
			uint64(len(onEvent)),
			func(timestamp uint64, payload []byte, offset uint64) error {
				e, err := generated.DecodeEventJSON(payload)
				require.NoError(s.t, err)
				check(time.Unix(int64(timestamp), 0), e)
				return nil
			},
		)
		require.NoError(s.t, err)
	}
}
