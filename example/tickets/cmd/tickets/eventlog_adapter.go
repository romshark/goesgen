package main

import (
	"context"
	"tickets/generated"
	"time"

	"github.com/romshark/eventlog/client"
	"github.com/romshark/eventlog/eventlog"
)

type EventlogAdapter struct {
	Client *client.Client
}

// IsOffsetOutOfBoundErr returns true if the given error
// is an offset-out-of-bound error
func (a *EventlogAdapter) IsOffsetOutOfBoundErr(err error) bool {
	return err == eventlog.ErrOffsetOutOfBound
}

// Begin returns the first offset version of the eventlog.
//
// WARNING: Begin is expected to be thread-safe.
func (a *EventlogAdapter) Begin(ctx context.Context) (string, error) {
	return a.Client.Begin(ctx)
}

// Scan reads a limited number of events at the given offset version
// calling the onEvent callback for every received event.
//
// WARNING: Scan is expected to be thread-safe.
func (a *EventlogAdapter) Scan(
	ctx context.Context,
	version generated.EventlogVersion,
	limit uint,
	onEvent func(
		offset generated.EventlogVersion,
		tm time.Time,
		payload []byte,
		next generated.EventlogVersion,
	) error,
) error {
	return a.Client.Scan(
		ctx,
		version,
		limit,
		func(
			offset string,
			tm time.Time,
			payload []byte,
			next string,
		) error {
			return onEvent(offset, tm, payload, next)
		},
	)
}

// AppendJSON appends one or multiple new events
// in JSON format onto the log.
//
// WARNING: AppendJSON is expected to be thread-safe.
func (a *EventlogAdapter) AppendJSON(
	ctx context.Context,
	payload []byte,
) (
	offset generated.EventlogVersion,
	newVersion generated.EventlogVersion,
	tm time.Time,
	err error,
) {
	return a.Client.AppendJSON(ctx, payload)
}

// TryAppendJSON keeps executing transaction until either cancelled,
// succeeded (assumed and actual event log versions match)
// or failed due to an error.
//
// WARNING: TryAppendJSON is expected to be thread-safe.
func (a *EventlogAdapter) TryAppendJSON(
	ctx context.Context,
	assumedVersion generated.EventlogVersion,
	transaction func() (events []byte, err error),
	sync func() (generated.EventlogVersion, error),
) (
	offset generated.EventlogVersion,
	newVersion generated.EventlogVersion,
	tm time.Time,
	err error,
) {
	return a.Client.TryAppendJSON(
		ctx, assumedVersion,
		func() (events []byte, err error) { return transaction() },
		func() (string, error) { return sync() },
	)
}
