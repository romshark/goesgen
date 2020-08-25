package gen

const TmplProjections = `{{define "projections"}}
/* PROJECTIONS */

{{range $n, $p := $.Schema.Projections}}
{{with $projName := $.ProjectionType $n}}

type {{$projName}}State string

const (
	{{range $sn, $s := $p.States}}
	{{$projName}}State{{$sn}} {{$projName}}State = "{{$sn}}"
	{{end}}
)

type {{$projName}} struct {
	state {{$projName}}State
}

func (p {{$projName}}) State() {{$projName}}State {
	return p.state
}

{{end}}
{{end}}

{{end}}
`
