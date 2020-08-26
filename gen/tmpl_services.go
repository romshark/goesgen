package gen

const TmplServices = `{{define "services"}}
/* SERVICES */

type EventlogVersion = string

type EventLog struct {
	IsOffsetOutOfBoundErr func(error) bool
	Logger                EventLogger
}

type EventLogger interface {
	// Listen starts listening for version update notifications
	// calling onUpdate when one is received.
	Listen(ctx context.Context, onUpdate func([]byte)) error

	// Scan reads a limited number of events at the given offset version
	// calling the onEvent callback for every received event.
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

{{range $n, $s := $.Schema.Services}}
{{with $srvName := $.ServiceType $n}}

// {{$srvName}} projects the following entities:
{{range $p := $s.Projections}}//  {{$p.Name}}{{end}}
// therefore, {{$srvName}} subscribes to the following events:
{{range $p := $s.Projections}}{{range $e, $t := $p.Transitions}}//  {{$e.Name}}
{{end}}{{end}}type {{$srvName}} struct {
	eventlog EventLog
	logErr   Logger

	lock              sync.Mutex
	projectionVersion EventlogVersion
	impl              {{$srvName}}Impl
}

// {{$srvName}}Impl represents the implementation of the service {{$srvName}}
type {{$srvName}}Impl interface {
	// ProjectionVersion returns the current projection version.
	ProjectionVersion(context.Context) (EventlogVersion, error)
	
	{{range $e := $s.Subscriptions}}
	// Apply{{$e.Name}} applies event {{$e.Name}} to the projection.
	Apply{{$.EventType $e.Name}} (time.Time, {{$.EventType $e.Name}})
	{{end}}

	{{range $mn, $m := $s.Methods}}
	// {{$.MethodName $mn}} represents method {{$srvName}}.{{$mn}}
	// 
	// WARNING: this method shall not affect the state of the projection.
	{{$.MethodName $mn}}(
		context.Context,
		{{if $m.Input}}src.{{$m.Input.Name}}, {{end}}
	) (
		{{if $m.Output}}src.{{$m.Output.Name}},{{end}}
		{{if (not (eq $m.Type "readonly"))}}[]Event,{{end}}
		error,
	)
	{{end}}
}

func New{{$srvName}}(
	implementation {{$srvName}}Impl,
	eventlog EventLog,
	logErr Logger,
) *{{$srvName}} {
	if implementation == nil {
		panic("implementation is nil in New{{$srvName}}")
	}
	if eventlog.IsOffsetOutOfBoundErr == nil {
		panic("eventlog.IsOffsetOutOfBoundErr is nil in New{{$srvName}}")
	}
	if eventlog.Logger == nil {
		panic("eventlog.Logger is nil in New{{$srvName}}")
	}
	if logErr == nil {
		logErr = defaultLogErr
	}
	return &{{$srvName}}{
		impl:                  implementation,
		eventlog:              eventlog,
		logErr:                logErr,
	}
}

func (s *{{$srvName}}) initialize(
	ctx context.Context,
) error {
	if s.projectionVersion != "" {
		return nil
	}
	v, err := s.impl.ProjectionVersion(ctx)
	if err != nil {
		return err
	}
	s.projectionVersion = v
	return nil
}

// Listen starts listening for updates asynchronously
// by subscribing to the eventlog's update notifier endpoint.
// Listen will block until the provided context is canceled.
func (s *{{$srvName}}) Listen(ctx context.Context) error {
	return s.eventlog.Logger.Listen(ctx, func(version []byte) {
		s.lock.Lock()
		defer s.lock.Unlock()

		if _, err := s.Sync(ctx); err != nil {
			s.logErr.Printf("async syncing: %s", err)
		}
	})
}

// Sync synchronizes {{$srvName}} against the eventlog
func (s *{{$srvName}}) Sync(
	ctx context.Context,
) (v EventlogVersion, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if err := s.initialize(ctx); err != nil {
		return "", err
	}

	if err := s.eventlog.Logger.Scan(
		ctx,
		s.projectionVersion,
		0,
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
			{{range $e := $s.Subscriptions}} case {{ $.EventType $e.Name }}:
				s.impl.Apply{{ $.EventType $e.Name }}(tm, v)
			{{end}}
			s.projectionVersion = next
			}
			return nil
		},
	); err != nil {
		if s.eventlog.IsOffsetOutOfBoundErr(err) {
			return s.projectionVersion, nil
		}
		return "", err
	}
	return s.projectionVersion, nil
}

// ProjectionVersion lazily initializes the service and
// returns the current projection version of the service
func (s *{{$srvName}}) ProjectionVersion(
	ctx context.Context,
) (EventlogVersion, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if err := s.initialize(ctx); err != nil {
		return "", err
	}
	return s.projectionVersion, nil
}

{{range $mn, $m := $s.Methods}}
func (s *{{$srvName}}) {{$mn}}(
	ctx context.Context,
	{{if $m.Input}}in src.{{$m.Input.Name}}, {{end}}
) (
	{{if $m.Output}}out src.{{$m.Output.Name}}, {{end}}
	{{if (not (eq $m.Type "readonly"))}}
	events []Event,
	eventsPushTime time.Time,
	{{end}}
	err error,
) {
	{{if $m.Output}}var outZero src.{{$m.Output.Name}}{{end}}
	{{if (not (eq $m.Type "readonly"))}}var eventsJSON []byte{{end}}
	defer func() {
		if err != nil {
			{{if $m.Output}}out = outZero{{end}}
			{{if (not (eq $m.Type "readonly"))}}
			events = nil
			eventsJSON = nil
			eventsPushTime = time.Time{}
			{{end}}
		}
	}()

	exec := func() (ok bool) {

		{{if $m.Output}}out, {{end}}
		{{if (not (eq $m.Type "readonly"))}}events, {{end}}
		err = s.impl.{{$.MethodName $mn}}(
			ctx,
			{{if $m.Input}}in,{{end}}
		)
		if err != nil {
			return false
		}
		{{if (not (eq $m.Type "readonly"))}}for i, e := range events {
			if err = CheckEventType(e); err != nil {
				err = fmt.Errorf("checking returned event (%d): %w", i, err)
				return false
			}
			switch e.(type) {
			{{range $e := $m.Emits}}case {{ $.EventType $e.Name}}:
			{{end}}
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
		{{end}}
		return true
	}

	{{if eq $m.Type "append"}}
	if !exec() {
		return
	}
	_, _, eventsPushTime, err = s.eventlog.Logger.AppendJSON(ctx, eventsJSON)
	{{else if eq $m.Type "transaction"}}
	s.lock.Lock()
	defer s.lock.Unlock()

	if err = s.initialize(ctx); err != nil {
		return
	}

	_, _, eventsPushTime, err = s.eventlog.Logger.TryAppendJSON(
		ctx,
		s.projectionVersion,
		func() ([]byte, error) {
			if !exec() {
				return nil, err
			}
			return eventsJSON, nil
		},
		func() (EventlogVersion, error) { return s.Sync(ctx) },
	)
	{{else}}
	exec()
	{{end}}

	return
}
{{end}}

{{end}}
{{end}}

{{end}}
`
