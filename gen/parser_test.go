package gen_test

import (
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/romshark/goesgen/gen"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	r := require.New(t)

	root, files := Setup(t, ValidSetup)

	s, err := gen.Parse(
		filepath.Join(root, "src"),
		files["schema.yaml"],
	)
	r.NoError(err)
	r.NotNil(s)

	{ // Check referenced types
		r.Len(s.SourcePackages, 3)

		{ // Check source package "src"
			const id = "src"
			r.Contains(s.SourcePackages, id)
			p := s.SourcePackages[id]
			r.Equal("src", p.Name)
			r.Equal(id, p.ID)
			r.Len(p.Types, 1)

			{ // Check src.Foo
				const id = "src.Foo"
				r.Contains(p.Types, id)
				t := p.Types[id]
				r.Equal("Foo", t.Name)
				r.Equal(id, t.ID)
				r.Equal(s.SourcePackages["src"], t.Package)
				r.Equal(
					token.Position{
						Filename: files["src/src.go"],
						Line:     2,
						Column:   6,
						Offset:   17,
					},
					t.SourceLocation,
				)
			}
		}

		{ // Check source package "src/sub"
			const id = "src.sub"
			r.Contains(s.SourcePackages, id)
			p := s.SourcePackages[id]
			r.Equal("sub", p.Name)
			r.Equal(id, p.ID)
			r.Len(p.Types, 1)

			{ // Check src.sub.Bar
				const id = "src.sub.Bar"
				r.Contains(p.Types, id)
				t := p.Types[id]
				r.Equal("Bar", t.Name)
				r.Equal(id, t.ID)
				r.Equal(s.SourcePackages["src.sub"], t.Package)
				r.Equal(
					token.Position{
						Filename: files["src/sub/sub.go"],
						Line:     2,
						Column:   6,
						Offset:   17,
					},
					t.SourceLocation,
				)
			}
		}

		{ // Check source package "src/sub/subsub"
			const id = "src.sub.subsub"
			r.Contains(s.SourcePackages, id)
			p := s.SourcePackages[id]
			r.Equal("subsub", p.Name)
			r.Equal(id, p.ID)
			r.Len(p.Types, 1)

			{ // Check src.Baz
				const id = "src.sub.subsub.Baz"
				r.Contains(p.Types, id)
				t := p.Types[id]
				r.Equal("Baz", t.Name)
				r.Equal(id, t.ID)
				r.Equal(s.SourcePackages["src.sub.subsub"], t.Package)
				r.Equal(
					token.Position{
						Filename: files["src/sub/subsub/subsub.go"],
						Line:     2,
						Column:   6,
						Offset:   20,
					},
					t.SourceLocation,
				)
			}
		}
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
		r.Equal(e.Properties[0].Name, "foo")
		r.Equal(e.Properties[0].Position, 0)
		CheckType(t, s, e.Properties[0].Type)
	}

	{ // events.E2
		r.Contains(s.Events, "E2")
		e := s.Events["E2"]
		r.Equal(s, e.Schema)
		r.Equal("E2", e.Name)

		// events.E2.properties
		r.Len(e.Properties, 2)

		// events.E2.properties.bar
		r.Equal(e.Properties[0].Name, "bar")
		r.Equal(e.Properties[0].Position, 0)
		CheckType(t, s, e.Properties[0].Type)

		// events.E2.properties.baz
		r.Equal(e.Properties[1].Name, "baz")
		r.Equal(e.Properties[1].Position, 1)
		CheckType(t, s, e.Properties[1].Type)
	}

	{ // events.E3
		r.Contains(s.Events, "E3")
		e := s.Events["E3"]
		r.Equal(s, e.Schema)
		r.Equal("E3", e.Name)

		// events.E3.properties
		r.Len(e.Properties, 1)

		// events.E3.properties.maz
		r.Equal(e.Properties[0].Name, "maz")
		r.Equal(e.Properties[0].Position, 0)
		CheckType(t, s, e.Properties[0].Type)
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
				CheckType(t, s, m.Input)
				CheckType(t, s, m.Output)
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
				CheckType(t, s, m.Output)
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
	r.Equal(`semantic error: projections.P1.transitions: `+
		`undefined event ("E3")`, err.Error())
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

func withOpenFile(p string, cb func(*os.File) error) error {
	f, err := os.OpenFile(
		p,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		0777,
	)
	if err != nil {
		return err
	}
	defer f.Close()

	return cb(f)
}

func Setup(
	t *testing.T,
	files Files,
) (root string, paths map[string]string) {
	root = t.TempDir()
	paths = make(map[string]string, len(files))
	for filePath, contents := range files {
		if dir, _ := filepath.Split(filePath); dir != "" {
			require.NoError(t, os.MkdirAll(
				filepath.Join(root, dir),
				0777,
			))
		}

		p := filepath.Join(root, filePath)
		err := withOpenFile(p, func(f *os.File) error {
			_, err := f.WriteString(contents)
			if err != nil {
				return err
			}

			paths[filePath] = p
			return nil
		})
		require.NoError(t, err)
	}
	return
}

type Files map[string]string

func CheckType(t *testing.T, s *gen.Schema, typ *gen.Type) {
	require.Contains(t, s.SourcePackages, typ.Package.ID)
	require.Contains(t, s.SourcePackages[typ.Package.ID].Types, typ.ID)
}

var ValidSetup = Files{
	"schema.yaml":              ValidSchemaSchemaYAML,
	"main.go":                  ValidSchemaMainGO,
	"src/src.go":               ValidSchemaSrcGO,
	"src/sub/sub.go":           ValidSchemaSubGO,
	"src/sub/subsub/subsub.go": ValidSchemaSubsubGO,
	"go.mod":                   ValidSchemaGoMOD,
}

const ValidSchemaSrcGO = `package src
type Foo = string
`

const ValidSchemaSubGO = `package sub
type Bar int
`

const ValidSchemaSubsubGO = `package subsub
type Baz struct { Number float64 }
`

const ValidSchemaSchemaYAML = `
---
events:
  E1:
    # foo defines foo
    foo: Foo
  E2:
    bar: sub.Bar
    # baz represents baz
    # 
    # and another ## comment line
    baz: sub.subsub.Baz
  E3:
    maz: Foo
projections:
  P1:
    properties:
      prop1: Foo
      prop2: sub.subsub.Baz
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
      # M1 does something
      M1:
        in: Foo
        out: sub.Bar
        type: transaction
        emits:
          - E1
      # M2 does something
      #
      # another line
      M2:
        type: append
        emits:
          - E2
          - E3
      M3:
        out: sub.subsub.Baz
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
