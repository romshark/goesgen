package gen_test

import (
	"go/token"
	"os"
	"path"
	"testing"

	"github.com/romshark/goesgen/gen"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	r := require.New(t)

	root, files := Setup(t, Files{
		"schema.yaml":      ValidSchemaSchemaYAML,
		"main.go":          ValidSchemaMainGO,
		"domain/domain.go": ValidSchemaDomainGO,
		"go.mod":           ValidSchemaGoMOD,
	})

	s, err := gen.Parse(
		path.Join(root, "domain"),
		files["schema.yaml"],
	)
	r.NoError(err)
	r.NotNil(s)

	// Check referenced types
	r.Len(s.ReferencedTypes, 3)
	for _, x := range []struct {
		name gen.TypeName
		pos  token.Position
	}{
		{"Foo", token.Position{
			Filename: files["domain/domain.go"],
			Line:     4,
			Column:   2,
			Offset:   24,
		}},
		{"Bar", token.Position{
			Filename: files["domain/domain.go"],
			Line:     5,
			Column:   2,
			Offset:   38,
		}},
		{"Baz", token.Position{
			Filename: files["domain/domain.go"],
			Line:     6,
			Column:   2,
			Offset:   47,
		}},
	} {
		r.Contains(s.ReferencedTypes, x.name)
		t := s.ReferencedTypes[x.name]
		r.Equal(x.name, t.Name)
		r.Equal(
			x.pos, t.SourceLocation,
			"unexpected location for type %s", x.name,
		)
	}

	// events
	r.Len(s.Events, 3)

	{ // events.E1
		r.Contains(s.Events, "E1")
		e := s.Events["E1"]
		r.Equal(s, e.Schema)
		r.Equal("E1", e.Name)

		// events.E1.properties
		r.Len(e.Properties, 1)

		// events.E1.properties.foo
		r.Contains(e.Properties, "foo")
		r.Equal(s.ReferencedTypes["Foo"], e.Properties["foo"])
	}

	{ // events.E2
		r.Contains(s.Events, "E2")
		e := s.Events["E2"]
		r.Equal(s, e.Schema)
		r.Equal("E2", e.Name)

		// events.E2.properties
		r.Len(e.Properties, 2)

		// events.E2.properties.bar
		r.Contains(e.Properties, "bar")
		r.Equal(s.ReferencedTypes["Bar"], e.Properties["bar"])

		// events.E2.properties.baz
		r.Contains(e.Properties, "baz")
		r.Equal(s.ReferencedTypes["Baz"], e.Properties["baz"])
	}

	{ // events.E3
		r.Contains(s.Events, "E3")
		e := s.Events["E3"]
		r.Equal(s, e.Schema)
		r.Equal("E3", e.Name)

		// events.E3.properties
		r.Len(e.Properties, 1)

		// events.E3.properties.maz
		r.Contains(e.Properties, "maz")
		r.Equal(s.ReferencedTypes["Foo"], e.Properties["maz"])
	}

	{ // projections
		r.Len(s.Projections, 1)

		// projections
		r.Contains(s.Projections, "P1")
		p := s.Projections["P1"]
		r.Equal("P1", p.Name)

		// projections.P1.createOn
		r.Equal("E1", p.CreateOn.Name)

		// projections.P1.states
		r.Equal(map[gen.ProjectionState]struct{}{
			"ST1": {},
			"ST2": {},
			"ST3": {},
		}, p.States)

		// projections.P1.transitions
		r.Len(p.Transitions, 2)

		{ // projections.P1.transitions.E2
			e := s.Events["E2"]
			r.Contains(p.Transitions, e)

			r.Len(p.Transitions[e], 1)
			{ // projections.P1.transitions.E2.0
				t := p.Transitions[e][0]
				r.Equal(p, t.Projection)
				r.Equal(gen.ProjectionState("ST1"), t.From)
				r.Equal(gen.ProjectionState("ST2"), t.To)
				r.Equal(e, t.On)
			}
		}

		{ // projections.P1.transitions.E3
			e := s.Events["E3"]
			r.Contains(p.Transitions, e)

			r.Len(p.Transitions[e], 2)

			{ // projections.P1.transitions.E3.0
				t := p.Transitions[e][0]
				r.Equal(p, t.Projection)
				r.Equal(gen.ProjectionState("ST2"), t.From)
				r.Equal(gen.ProjectionState("ST2"), t.To)
				r.Equal(e, t.On)
			}

			{ // projections.P1.transitions.E3.1
				t := p.Transitions[e][1]
				r.Equal(p, t.Projection)
				r.Equal(gen.ProjectionState("ST3"), t.From)
				r.Equal(gen.ProjectionState("ST3"), t.To)
				r.Equal(e, t.On)
			}
		}
	}

	{ // services
		r.Len(s.Services, 1)

		{ // services.S1
			r.Contains(s.Services, "S1")
			service := s.Services["S1"]
			r.Equal(s, service.Schema)
			r.Equal("S1", service.Name)

			{ // services.S1.subscriptions
				r.Len(service.Subscriptions, 3)

				// services.S1.subscriptions.E1
				r.Contains(service.Subscriptions, "E1")
				r.Equal(s.Events["E1"], service.Subscriptions["E1"])

				// services.S1.subscriptions.E2
				r.Contains(service.Subscriptions, "E2")
				r.Equal(s.Events["E2"], service.Subscriptions["E2"])

				// services.S1.subscriptions.E3
				r.Contains(service.Subscriptions, "E3")
				r.Equal(s.Events["E3"], service.Subscriptions["E3"])
			}

			// services.S1.methods
			r.Len(service.Methods, 5)

			{ // services.S1.methods.M1
				m := service.Methods["M1"]
				r.Equal(s.ReferencedTypes["Foo"], m.Input)
				r.Equal(s.ReferencedTypes["Bar"], m.Output)
				r.Equal("M1", m.Name)
				r.Equal(gen.ServiceMethodType("transaction"), m.Type)
				r.Equal(
					[]*gen.Event{
						s.Events["E1"],
					},
					m.Emits,
				)
			}

			{ // services.S1.methods.M2
				m := service.Methods["M2"]
				r.Equal("M2", m.Name)
				r.Equal(gen.ServiceMethodType("append"), m.Type)
				r.Nil(m.Input)
				r.Nil(m.Output)
				r.Equal(
					[]*gen.Event{
						s.Events["E2"],
						s.Events["E3"],
					},
					m.Emits,
				)
			}

			{ // services.S1.methods.M3
				m := service.Methods["M3"]
				r.Equal("M3", m.Name)
				r.Equal(gen.ServiceMethodType("readonly"), m.Type)
				r.Nil(m.Input)
				r.Equal(s.ReferencedTypes["Baz"], m.Output)
				r.Len(m.Emits, 0)
			}

			{ // services.S1.methods.M4
				m := service.Methods["M4"]
				r.Equal("M4", m.Name)
				r.Equal(gen.ServiceMethodType("readonly"), m.Type)
				r.Nil(m.Input)
				r.Nil(m.Output)
				r.Len(m.Emits, 0)
			}

			{ // services.S1.methods.M5
				m := service.Methods["M5"]
				r.Equal("M5", m.Name)
				r.Equal(gen.ServiceMethodType("transaction"), m.Type)
				r.Nil(m.Input)
				r.Nil(m.Output)
				r.Equal(
					[]*gen.Event{
						s.Events["E3"],
					},
					m.Emits,
				)
			}
		}
	}
}

func TestParseUndeclaredEvent(t *testing.T) {
	root, files := Setup(t, Files{
		"schema.yaml": `
---
events:
  E1:
    foo: Foo
  E2:
    bar: Bar
    baz: Baz
projections:
  P1:
    properties:
      prop1: Foo
      prop2: Baz
    states:
      - ST1
      - ST2
      - ST3
    createOn: E1
    transitions:
      E2:
        - ST1 -> ST2
      E3:
        - ST2 -> ST2
        - ST3 -> ST3
services:
  S1:
    projections:
      - P1
    methods:
      M1:
        emits:
          - E1
`,

		"source.go": `package main

type (
	Foo = string
	Bar int
	Baz struct {
		Number float64
	}
)
`,

		"go.mod": `module tst
		
go 1.15`,
	})

	schema, err := gen.Parse(root, files["schema.yaml"])
	r := require.New(t)
	r.Error(err)
	r.IsType(gen.SemanticErr(""), err, err.Error())
	r.Equal(`semantic error: undefined event ("E3") `+
		`in projections.P1.transitions`, err.Error())
	r.Nil(schema)
}

func TestParseUnusedEvent(t *testing.T) {
	root, files := Setup(t, Files{
		"schema.yaml": `
---
events:
  E1:
    foo: T
  E2:
    bar: T
projections:
  P1:
    properties:
      prop1: T
    states:
      - ST1
    createOn: E1
services:
  S1:
    methods:
      M1:
        in: T
`,
		"main.go": `package main; type T = int`,
	})

	schema, err := gen.Parse(root, files["schema.yaml"])
	r := require.New(t)
	r.Error(err)
	r.IsType(gen.SemanticErr(""), err, err.Error())
	r.Equal(`semantic error: unused event (E2)`, err.Error())
	r.Nil(schema)
}

func Setup(
	t *testing.T,
	files Files,
) (root string, paths map[string]string) {
	root = t.TempDir()
	paths = make(map[string]string, len(files))
	for filePath, contents := range files {
		if dir, _ := path.Split(filePath); dir != "" {
			require.NoError(t, os.MkdirAll(
				path.Join(root, dir),
				0777,
			))
		}

		p := path.Join(root, filePath)
		f, err := os.OpenFile(
			p,
			os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
			0777,
		)
		require.NoError(t, err)
		defer f.Close()
		_, err = f.WriteString(contents)
		require.NoError(t, err)
		paths[filePath] = p
	}
	return
}

type Files map[string]string

const ValidSchemaDomainGO = `package domain

type (
	Foo = string
	Bar int
	Baz struct {
		Number float64
	}
)
`

const ValidSchemaSchemaYAML = `
---
events:
  E1:
    foo: Foo
  E2:
    bar: Bar
    baz: Baz
  E3:
    maz: Foo
projections:
  P1:
    properties:
      prop1: Foo
      prop2: Baz
    states:
      - ST1
      - ST2
      - ST3
    createOn: E1
    transitions:
      E2:
        - ST1 -> ST2
      E3:
        - ST2 -> ST2
        - ST3 -> ST3
services:
  S1:
    projections:
      - P1
    methods:
      M1:
        in: Foo
        out: Bar
        type: transaction
        emits:
          - E1
      M2:
        type: append
        emits:
          - E2
          - E3
      M3:
        out: Baz
      M4:
        type: readonly
      M5:
        emits:
          - E3
`

const ValidSchemaGoMOD = `module testmod

go 1.15`

const ValidSchemaMainGO = `package main

func main() {}
`
