{{define "projections"}}
/* PROJECTIONS */

{{range $n, $p := $.Schema.Projections}}
{{with $projType := $.ProjectionType $n}}

type {{$projType}}State string

const (
	{{- range $sn, $s := $p.States}}
	{{$.ProjectionStateConstant $projType $sn}} {{$projType}}State = "{{$sn}}"
	{{- end -}}
)

type {{$projType}} struct {
	state {{$projType}}State
}

func New{{$projType}}() {{$projType}} {
	return {{$projType}}{
		state: {{$.ProjectionStateConstant $projType $p.InitialState}},
	}
}

func (p {{$projType}}) State() {{$projType}}State {
	return p.state
}

{{end}}
{{end}}

{{end}}
