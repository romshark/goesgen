package gen

const TmplEvents = `{{define "events"}}
/* EVENT TYPES */

// Event represents either of:
{{range $n, $e := $.Schema.Events}}//  {{$.EventType $n}}
{{end -}}
type Event = interface{}

{{range $n, $e := $.Schema.Events}}
// {{$.EventType $n}} defines event {{$n}}
type {{$.EventType $n}} struct {
	{{range $p := $e.Properties -}}
	{{range $l := $p.CommentLines}}
	// {{$l}}
	{{- end}}
	{{$.Capitalize $p.Name}} {{$.TypeID $p.Type}} "json:\"{{$p.Name}}\""
	{{end -}}
}
{{end}}

// GetEventTypeName returns the given event's name.
// Returns "" if the given object is not a valid event.
func GetEventTypeName(e Event) string {
	switch e.(type) {
	{{- range $n := $.Schema.Events}}
	case {{$.EventType $n.Name}}: return "{{$n.Name}}"
	{{- end}}
	}
	return ""
}

// CheckEventType returns an error if the given object isn't a valid event,
// otherwise returns nil.
func CheckEventType(e Event) error {
	if GetEventTypeName(e) == "" {
		return UnknownEventTypeErr(fmt.Sprintf(
			"unknown event type %s", reflect.TypeOf(e),
		))
	}
	return nil
}

{{end}}
`
