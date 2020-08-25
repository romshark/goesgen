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
		"go.sum":           ValidSchemaGoSUM,
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

go 1.15

require github.com/romshark/eventlog v0.0.0-20200807013727-d8e66a81e930
`

const ValidSchemaGoSUM = `
github.com/OneOfOne/xxhash v1.2.2 h1:KMrpdQIwFcEqXDklaen+P1axHaj9BSKzvpUUfnHldSE=
github.com/OneOfOne/xxhash v1.2.2/go.mod h1:HSdplMjZKSmBqAxg5vPj2TmRDmfkzw+cTzAElWljhcU=
github.com/andybalholm/brotli v1.0.0 h1:7UCwP93aiSfvWpapti8g88vVVGp2qqtGyePsSuDafo4=
github.com/andybalholm/brotli v1.0.0/go.mod h1:loMXtMfwqflxFJPmdbJO0a3KNoPuLBgiu3qAvBg8x/Y=
github.com/cespare/xxhash v1.1.0 h1:a6HrQnmkObjyL+Gs60czilIUGqrzKutQD6XZog3p+ko=
github.com/cespare/xxhash v1.1.0/go.mod h1:XrSqR1VqqWfGrhpAt58auRo0WTKS1nRRg3ghfAqPWnc=
github.com/davecgh/go-spew v1.1.0/go.mod h1:J7Y8YcW2NihsgmVo/mv3lAwl/skON4iLHjSsI+c5H38=
github.com/davecgh/go-spew v1.1.1 h1:vj9j/u1bqnvCEfJOwUhtlOARqs3+rkHYY13jYWTU97c=
github.com/davecgh/go-spew v1.1.1/go.mod h1:J7Y8YcW2NihsgmVo/mv3lAwl/skON4iLHjSsI+c5H38=
github.com/fasthttp/websocket v1.4.3 h1:qjhRJ/rTy4KB8oBxljEC00SDt6HUY9jLRfM601SUdS4=
github.com/fasthttp/websocket v1.4.3/go.mod h1:5r4oKssgS7W6Zn6mPWap3NWzNPJNzUUh3baWTOhcYQk=
github.com/klauspost/compress v1.10.4 h1:jFzIFaf586tquEB5EhzQG0HwGNSlgAJpG53G6Ss11wc=
github.com/klauspost/compress v1.10.4/go.mod h1:aoV0uJVorq1K+umq18yTdKaF57EivdYsUV+/s2qKfXs=
github.com/kr/pretty v0.1.0 h1:L/CwN0zerZDmRFUapSPitk6f+Q3+0za1rQkzVuMiMFI=
github.com/kr/pretty v0.1.0/go.mod h1:dAy3ld7l9f0ibDNOQOHHMYYIIbhfbHSm3C4ZsoJORNo=
github.com/kr/pty v1.1.1/go.mod h1:pFQYn66WHrOpPYNljwOMqo10TkYh1fy3cYio2l3bCsQ=
github.com/kr/text v0.1.0 h1:45sCR5RtlFHMR4UwH9sdQ5TC8v0qDQCHnXt+kaKSTVE=
github.com/kr/text v0.1.0/go.mod h1:4Jbv+DJW3UT/LiOwJeYQe1efqtUx/iVham/4vfdArNI=
github.com/pmezard/go-difflib v1.0.0 h1:4DBwDE0NGyQoBHbLQYPwSUPoCMWR5BEzIk/f1lZbAQM=
github.com/pmezard/go-difflib v1.0.0/go.mod h1:iKH77koFhYxTK1pcRnkKkqfTogsbg7gZNVY4sRDYZ/4=
github.com/romshark/eventlog v0.0.0-20200807013727-d8e66a81e930 h1:yBbwchhzw9TW+GOqG4PfQ9xI33RWXLA3tQT5lBZM5mQ=
github.com/romshark/eventlog v0.0.0-20200807013727-d8e66a81e930/go.mod h1:jgitd4vAPezlnZN3n4GN+MZoym/kLPpcMGHc3QYiOpU=
github.com/savsgio/gotils v0.0.0-20200608150037-a5f6f5aef16c h1:2nF5+FZ4/qp7pZVL7fR6DEaSTzuDmNaFTyqp92/hwF8=
github.com/savsgio/gotils v0.0.0-20200608150037-a5f6f5aef16c/go.mod h1:TWNAOTaVzGOXq8RbEvHnhzA/A2sLZzgn0m6URjnukY8=
github.com/spaolacci/murmur3 v0.0.0-20180118202830-f09979ecbc72 h1:qLC7fQah7D6K1B0ujays3HV9gkFtllcxhzImRR7ArPQ=
github.com/spaolacci/murmur3 v0.0.0-20180118202830-f09979ecbc72/go.mod h1:JwIasOWyU6f++ZhiEuf87xNszmSA2myDM2Kzu9HwQUA=
github.com/stretchr/objx v0.1.0/go.mod h1:HFkY916IF+rwdDfMAkV7OtwuqBVzrE8GR6GFx+wExME=
github.com/stretchr/testify v1.4.0 h1:2E4SXV/wtOkTonXsotYi4li6zVWxYlZuYNCXe9XRJyk=
github.com/stretchr/testify v1.4.0/go.mod h1:j7eGeouHqKxXV5pUuKE4zz7dFj8WfuZ+81PSLYec5m4=
github.com/tdewolff/minify v2.3.6+incompatible h1:2hw5/9ZvxhWLvBUnHE06gElGYz+Jv9R4Eys0XUzItYo=
github.com/tdewolff/minify v2.3.6+incompatible/go.mod h1:9Ov578KJUmAWpS6NeZwRZyT56Uf6o3Mcz9CEsg8USYs=
github.com/tdewolff/parse v2.3.4+incompatible h1:x05/cnGwIMf4ceLuDMBOdQ1qGniMoxpP46ghf0Qzh38=
github.com/tdewolff/parse v2.3.4+incompatible/go.mod h1:8oBwCsVmUkgHO8M5iCzSIDtpzXOT0WXX9cWhz+bIzJQ=
github.com/tdewolff/test v1.0.6 h1:76mzYJQ83Op284kMT+63iCNCI7NEERsIN8dLM+RiKr4=
github.com/tdewolff/test v1.0.6/go.mod h1:6DAvZliBAAnD7rhVgwaM7DE5/d9NMOAJ09SqYqeK4QE=
github.com/valyala/bytebufferpool v1.0.0 h1:GqA5TC/0021Y/b9FG4Oi9Mr3q7XYx6KllzawFIhcdPw=
github.com/valyala/bytebufferpool v1.0.0/go.mod h1:6bBcMArwyJ5K/AmCkWv1jt77kVWyCJ6HpOuEn7z0Csc=
github.com/valyala/fasthttp v1.14.0 h1:67bfuW9azCMwW/Jlq/C+VeihNpAuJMWkYPBig1gdi3A=
github.com/valyala/fasthttp v1.14.0/go.mod h1:ol1PCaL0dX20wC0htZ7sYCsvCYmrouYra0zHzaclZhE=
github.com/valyala/fastjson v1.4.5 h1:uSuLfXk2LzRtzwd3Fy5zGRBe0Vs7zhs11vjdko32xb4=
github.com/valyala/fastjson v1.4.5/go.mod h1:nV6MsjxL2IMJQUoHDIrjEI7oLyeqK6aBD7EFWPsvP8o=
github.com/valyala/tcplisten v0.0.0-20161114210144-ceec8f93295a/go.mod h1:v3UYOV9WzVtRmSR+PDvWpU/qWl4Wa5LApYYX4ZtKbio=
golang.org/x/crypto v0.0.0-20190308221718-c2843e01d9a2/go.mod h1:djNgcEr1/C05ACkg1iLfiJU5Ep61QUkGW8qpdssI0+w=
golang.org/x/net v0.0.0-20200324143707-d3edc9973b7e/go.mod h1:qpuaurCH72eLCgpAm/N6yyVIVM9cpaDIP3A8BGJEC5A=
golang.org/x/sys v0.0.0-20190215142949-d0b11bdaac8a/go.mod h1:STP8DvDyc/dI5b8T5hshtkjS+E42TnysNCUPdjciGhY=
golang.org/x/sys v0.0.0-20200323222414-85ca7c5b95cd/go.mod h1:h1NjWce9XRLGQEsW7wpKNCjG9DtNlClVuFLEZdDNbEs=
golang.org/x/text v0.3.0/go.mod h1:NqM8EUOU14njkJ3fqMW+pc6Ldnwhi/IjpwHt7yyuwOQ=
gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405/go.mod h1:Co6ibVJAznAaIkqp8huTwlJQCZ016jof/cbN4VW5Yz0=
gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 h1:qIbj1fsPNlZgppZ+VLlY7N33q108Sa+fhmuc+sWQYwY=
gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127/go.mod h1:Co6ibVJAznAaIkqp8huTwlJQCZ016jof/cbN4VW5Yz0=
gopkg.in/yaml.v2 v2.2.2 h1:ZCJp+EgiOT7lHqUV2J862kp8Qj64Jo6az82+3Td9dZw=
gopkg.in/yaml.v2 v2.2.2/go.mod h1:hI93XBmqTisBFMUTm0b8Fm+jr3Dg1NNxqwp+5A1VGuI=
`

const ValidSchemaMainGO = `package main

func main() {}
`
