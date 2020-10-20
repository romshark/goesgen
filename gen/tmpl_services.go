package gen

const TmplServices = `{{define "services"}}
/* SERVICES */

type EventlogVersion = string

// EventLogger represents an abstract event logger
type EventLogger interface {
	// IsOffsetOutOfBoundErr returns true if the given error
	// is an offset-out-of-bound error
	IsOffsetOutOfBoundErr(error) bool

	// Begin returns the first offset version of the eventlog.
	//
	// WARNING: Begin is expected to be thread-safe.
	Begin(context.Context) (string, error)

	// Scan reads a limited number of events at the given offset version
	// calling the onEvent callback for every received event.
	//
	// WARNING: Scan is expected to be thread-safe.
	Scan(
		ctx context.Context,
		version EventlogVersion,
		limit uint,
		onEvent func(
			offset EventlogVersion,
			tm time.Time,
			payload []byte,
			next EventlogVersion,
		) error,
	) error

	// AppendJSON appends one or multiple new events
	// in JSON format onto the log.
	//
	// WARNING: AppendJSON is expected to be thread-safe.
	AppendJSON(
		ctx context.Context,
		payload []byte,
	) (
		offset EventlogVersion,
		newVersion EventlogVersion,
		tm time.Time,
		err error,
	)

	// TryAppendJSON keeps executing transaction until either cancelled,
	// succeeded (assumed and actual event log versions match)
	// or failed due to an error.
	//
	// WARNING: TryAppendJSON is expected to be thread-safe.
	TryAppendJSON(
		ctx context.Context,
		assumedVersion EventlogVersion,
		transaction func() (events []byte, err error),
		sync func() (EventlogVersion, error),
	) (
		offset EventlogVersion,
		newVersion EventlogVersion,
		tm time.Time,
		err error,
	)
}

// StoreTransactionReadWriter represents an abstract
// read-write (exclusive locking) store transaction handler
type StoreTransactionReadWriter interface {
	Commit()
	Rollback()
}

// StoreTransactionReader represents an abstract
// read only (shared locking) store transaction handler
type StoreTransactionReader interface {
	Complete()
}

// TransactionWriter represents an arbitrary abstract transaction object
// that's supposed to be used for write-only mutations.
// TransactionWriter must not be committed or rolled back!
type TransactionWriter = interface{}

// TransactionReader represents an arbitrary abstract transaction object
// that's supposed to be used for read-only queries.
// TransactionReader must not be committed or rolled back!
type TransactionReader = interface{}

{{range $srvName, $s := $.Schema.Services}}
{{with $srvType := $.ServiceType $srvName}}

// {{$srvType}} projects the following entities:
{{range $p := $s.Projections}}//  {{$p.Name}}{{end}}
// therefore, {{$srvName}} subscribes to the following events:
{{range $p := $s.Projections}}{{range $e, $t := $p.Transitions}}//  {{$e.Name}}
{{end}}{{end -}}
type {{$srvType}} struct {
	eventlog EventLogger
	logErr   Logger
	methods  {{$srvType}}MethodCaller
	store    {{$srvType}}StoreHandler
}

// {{$srvType}}StoreHandler represents a store handler implementation
// of the service {{$srvName}}
type {{$srvType}}StoreHandler interface {
	// NewTransactionReadWriter creates a new exclusive
	// read-write transaction handler.
	// The returned transaction is passed to implementation methods
	// and will eventually be either committed or rolled back respectively.
	NewTransactionReadWriter() StoreTransactionReadWriter

	// NewTransactionReader creates a new read-only transaction handler.
	// The returned transaction is passed to implementation methods
	// and will eventually be completed.
	NewTransactionReader() StoreTransactionReader

	// ProjectionVersion returns the current projection version.
	// Returns an empty string if the projection wasn't initialized yet.
	// In case an empty string is returned the service will fallback
	// to the begin offset version of the eventlog.
	ProjectionVersion(
		context.Context,
		TransactionReader,
	) (EventlogVersion, error)

	// UpdateProjectionVersion explicitly sets the version of the store
	UpdateProjectionVersion(
		context.Context,
		TransactionWriter,
		EventlogVersion,
	) error
	
	{{range $e := $s.Subscriptions}}
	// Apply{{$.EventType $e.Name}} applies event {{$e.Name}} to the projection.
	// The projection must update its local projection version
	// to the one that is provided.
	Apply{{$.EventType $e.Name}} (
		context.Context,
		TransactionWriter,
		EventlogVersion,
		time.Time,
		{{$.EventType $e.Name}},
	) error
	{{end}}
}

// {{$srvType}}MethodCaller represents an implementation
// of the service {{$srvName}}
type {{$srvType}}MethodCaller interface {
	{{range $mn, $m := $s.Methods}}
	// {{$mn}} represents method {{$srvName}}.{{$mn}}
	{{- range $l := $m.CommentLines}}
	// {{$l}}
	{{- end}}
	//
	// WARNING: this method is read-only and must not mutate neither
	// the state of the projection nor the projection version!
	// The provided transaction must not be committed or rolled back
	// and shall only be used for queries and mutations.
	{{$mn}}(
		context.Context,
		TransactionReader,
		{{if $m.Input -}}
		{{$.TypeID $m.Input}},
		{{- else -}}
		// No input
		{{- end}}
	) (
		{{if $m.Output -}}
		output {{$.TypeID $m.Output}},
		{{- else -}}
		// No output
		{{- end}}
		{{if (not (eq $m.Type "readonly")) -}}
		events []Event,
		{{- else -}}
		// No events
		{{- end}}
		err error,
	)
	{{end}}
}

// New{{$srvType}} creates a new instance of the {{$srvName}} service.
func New{{$srvType}}(
	methodCaller {{$srvType}}MethodCaller,
	storeHandler {{$srvType}}StoreHandler,
	eventLogger EventLogger,
	errorLogger Logger,
) *{{$srvType}} {
	if methodCaller == nil {
		panic("methodCaller is nil in New{{$srvType}}")
	}
	if storeHandler == nil {
		panic("storeHandler is nil in New{{$srvType}}")
	}
	if eventLogger == nil {
		panic("eventLogger is nil in New{{$srvType}}")
	}
	if errorLogger == nil {
		errorLogger = defaultLogErr
	}
	return &{{$srvType}}{
		methods:  methodCaller,
		store:    storeHandler,
		eventlog: eventLogger,
		logErr:   errorLogger,
	}
}

// ProjectionVersion returns the current projection version
func (s *{{$srvType}}) ProjectionVersion(ctx context.Context) (
	EventlogVersion,
	error,
) {
	txn := s.store.NewTransactionReader()
	defer txn.Complete()

	return s.projectionVersion(ctx, txn)
}

func (s *{{$srvType}}) projectionVersion(
	ctx context.Context,
	txn TransactionReader,
) (
	EventlogVersion,
	error,
) {
	v, err := s.store.ProjectionVersion(ctx, txn)
	if err != nil {
		return "", fmt.Errorf("reading projection version: %w", err)
	}
	if v != "" {
		return v, nil
	}

	// Fallback to the beginning of the eventlog
	v, err = s.eventlog.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("reading begin offset version: %w", err)
	}
	return v, nil
}

// Sync synchronizes service {{$srvName}} against the eventlog.
// Sync will scan events until it reaches the tip of the event log and
// always return the latest version of the event log it managed to reach,
// unless the returned error is not equal context.Canceled or
// context.DeadlineExceeded and didn't pass isErrAcceptable (if not nil).
func (s *{{$srvType}}) Sync(
	ctx context.Context,
	isErrAcceptable func(error) bool,
) (
	latestVersion EventlogVersion,
	err error,
) {
	txn := s.store.NewTransactionReadWriter()
	defer func() {
		if err == nil ||
			errors.Is(err, context.Canceled) ||
			errors.Is(err, context.DeadlineExceeded) ||
			(isErrAcceptable != nil && isErrAcceptable(err)) {
			txn.Commit()
		} else {
			txn.Rollback()
		}
	}()

	return s.sync(ctx, txn)
}

func (s *{{$srvType}}) sync(
	ctx context.Context,
	trx TransactionWriter,
) (
	latestVersion EventlogVersion,
	err error,
) {
	initialVersion, err := s.projectionVersion(ctx, trx)
	if err != nil {
		return "", err
	}

	var appliedVersion EventlogVersion
	defer func() {
		if appliedVersion == latestVersion {
			return
		}
		err = s.store.UpdateProjectionVersion(ctx, trx, latestVersion)
		if err != nil {
			latestVersion = appliedVersion
		}
	}()

	if err := s.eventlog.Scan(
		ctx,
		initialVersion,
		0, // No limit
		func(
			offset EventlogVersion,
			tm time.Time,
			payload []byte,
			next EventlogVersion,
		) error {
			ev, err := DecodeEventJSON(payload)
			if err != nil {
				return err
			}
			switch v := ev.(type) {
			{{- range $e := $s.Subscriptions}} case {{ $.EventType $e.Name }}:
				if err := s.store.Apply{{ $.EventType $e.Name }}(
					ctx, trx, next, tm, v,
				); err != nil {
					return err
				}
				if err := s.store.UpdateProjectionVersion(
					ctx, trx, next,
				); err != nil {
					return err
				}
				appliedVersion = next
			{{end -}}
			}
			latestVersion = next
			return nil
		},
	); err != nil {
		if s.eventlog.IsOffsetOutOfBoundErr(err) {
			return latestVersion, nil
		}
		return "", err
	}
	return latestVersion, nil
}

{{range $mn, $m := $s.Methods}}
{{- range $l := $m.CommentLines}}
// {{$l}}
{{- end}}
func (s *{{$srvType}}) {{$mn}}(
	ctx context.Context,
	{{if $m.Input -}}
	input {{$.TypeID $m.Input}},
	{{- else -}}
	// No input
	{{- end}}
) (
	{{if $m.Output -}}
	output {{$.TypeID $m.Output}},
	{{- else -}}
	// No output
	{{- end}}
	{{if (not (eq $m.Type "readonly")) -}}
	events []Event,
	eventsPushTime time.Time,
	{{- else -}}
	// No events
	{{- end}}
	err error,
) {
	{{- if eq $m.Type "transaction"}}
	txn := s.store.NewTransactionReadWriter()
	defer func() {
		if err == nil ||
			errors.Is(err, context.Canceled) ||
			errors.Is(err, context.DeadlineExceeded) {
			txn.Commit()
		} else {
			txn.Rollback()
		}
	}()
	{{else}}
	txn := s.store.NewTransactionReader()
	defer txn.Complete()
	{{end}}

	{{if $m.Output -}}
	var outZero {{$.TypeID $m.Output}}
	{{- end}}
	{{if (not (eq $m.Type "readonly")) -}}
	var eventsJSON []byte
	{{- end}}

	{{if (or $m.Output (not (eq $m.Type "readonly"))) -}}
	defer func() {
		if err != nil {
			{{if $m.Output -}}
			output = outZero
			{{- else -}}
			// No output to reset
			{{- end}}
			{{if (not (eq $m.Type "readonly")) -}}
			events = nil
			eventsJSON = nil
			eventsPushTime = time.Time{}
			{{- else -}}
			// No events to reset
			{{- end}}
		}
	}()
	{{- end}}

	exec := func() (ok bool) {
		{{if $m.Output -}}
		output,
		{{- end -}}
		{{if (not (eq $m.Type "readonly")) -}}
		events,
		{{- end -}}
		err = s.methods.{{$mn}}(ctx, txn,
			{{- if $m.Input -}}
			input,
			{{- end -}}
		)
		if err != nil {
			return false
		}
		{{if (not (eq $m.Type "readonly")) -}}
		for i, e := range events {
			if err = CheckEventType(e); err != nil {
				err = fmt.Errorf("checking returned event (%d): %w", i, err)
				return false
			}
			switch e.(type) {
			{{- range $e := $m.Emits -}}
			case {{ $.EventType $e.Name}}:
			{{end -}}
			default:
				panic(fmt.Errorf(
					"method {{$s.Name}}.{{$mn}} is not allowed to emit event %s",
					reflect.TypeOf(e),
				))
			}
		}
		if eventsJSON, err = EncodeEventJSON(events...); err != nil {
			return false
		}
		{{- end}}
		return true
	}

	{{if eq $m.Type "append" -}}
	if !exec() {
		return
	}
	_, _, eventsPushTime, err = s.eventlog.AppendJSON(ctx, eventsJSON)
	{{- else if eq $m.Type "transaction" -}}
	var currentVersion EventlogVersion
	currentVersion, err = s.projectionVersion(ctx, txn)
	if err != nil {
		return
	}

	_, _, eventsPushTime, err = s.eventlog.TryAppendJSON(
		ctx,
		currentVersion,
		func() ([]byte, error) {
			if !exec() {
				return nil, err
			}
			return eventsJSON, nil
		},
		func() (EventlogVersion, error) { return s.sync(ctx, txn) },
	)
	{{- else}}
	exec()
	{{- end}}

	return
}
{{end}}

{{end}}
{{end}}

{{end}}
`
