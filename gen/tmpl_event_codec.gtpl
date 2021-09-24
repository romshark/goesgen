{{define "event_codec"}}
/* EVENT CODEC */

// EncodeEventJSON encodes one or multiple events to UTF-8 text.
// Multiple events are automatically encoded into a JSON array.
func EncodeEventJSON(e ...Event) ([]byte, error) {
	type E struct {
		TypeName string "json:\"type\""
		Payload  Event  "json:\"payload\""
	}

	if len(e) < 1 {
		return nil, nil
	}
	if len(e) < 2 {
		e := e[0]
		if err := CheckEventType(e); err != nil {
			return nil, err
		}
		return json.Marshal(E{
			TypeName: GetEventTypeName(e),
			Payload:  e,
		})
	}
	m := make([]E, len(e))
	for i, e := range e {
		if err := CheckEventType(e); err != nil {
			return nil, err
		}
		m[i] = E{
			TypeName: GetEventTypeName(e),
			Payload:  e,
		}
	}
	return json.Marshal(m)
}

// DecodeEventJSON decodes an event from UTF-8 text
func DecodeEventJSON(b []byte) (Event, error) {
	var v struct {
		TypeName string          "json:\"type\""
		Payload  json.RawMessage "json:\"payload\""
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(&v); err != nil {
		return nil, DecodingEventErr(fmt.Sprintf("decoding event: %s", err))
	}

	switch v.TypeName {
	{{- range $e := $.Schema.Events}}
	case "{{$e.Name}}":
		var e Event{{$e.Name}}
		if err := json.Unmarshal(v.Payload, &e); err != nil {
			return nil, DecodingEventErr(fmt.Sprintf(
				"decoding {{$e.Name}} payload: %s",
				err,
			))
		}
		return e, nil
	{{- end}}
	}
	return nil, UnknownEventTypeErr(fmt.Sprintf(
		"unknown event type %s", v.TypeName,
	))
}

type DecodingEventErr string

func (e DecodingEventErr) Error() string { return string(e) }

type UnknownEventTypeErr string

func (e UnknownEventTypeErr) Error() string { return string(e) }

{{end}}
