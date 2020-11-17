package gen_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/romshark/goesgen/gen"

	"github.com/stretchr/testify/require"
)

func TestGeneratorOptionsPrepareErrPackageName(t *testing.T) {
	const (
		expectError   = true
		expectNoError = false
	)
	for _, t1 := range []struct {
		expectError bool
		name        string
		packageName string
	}{
		{expectNoError, "okay", "okay"},
		{expectError, "spaces", "contains spaces"},
		{expectError, "underscore", "contains_underscores"},
		{expectError, "illegal characters", "contains?#!-"},
		{expectError, "camel case", "camelCase"},
		{expectError, "non-ascii", "абвгд"},
	} {
		t.Run(t1.name, func(t *testing.T) {
			o := &gen.GeneratorOptions{
				PackageName: t1.packageName,
			}
			if t1.expectError {
				require.Error(t, o.Prepare())
			} else {
				require.NoError(t, o.Prepare())
			}
		})
	}
}

func GoTool(dir string, args ...string) (string, string, error) {
	gotool := "go"
	if v, ok := os.LookupEnv("GOBIN"); ok {
		gotool = v
	}

	cmd := exec.Command(gotool, args...)
	var stdOut strings.Builder
	var errOut strings.Builder
	cmd.Stdout = &stdOut
	cmd.Stderr = &errOut
	cmd.Dir = dir

	err := cmd.Run()
	return stdOut.String(), errOut.String(), err
}

func TestUserTemplates(t *testing.T) {
	r := require.New(t)

	root, files := Setup(t, ValidSetup)

	schema, err := gen.Parse(root, files["schema.yaml"])
	r.NoError(err)

	g := gen.NewGenerator()
	r.NotNil(g)

	tmpl := template.Must(template.New("generated").Parse(`
package main

import "fmt"

func main() {
	fmt.Println("Hello, goesgen!")
}
`))
	template.Must(tmpl.Parse(`{{ define "events" }}{{ end -}}`))
	template.Must(tmpl.Parse(`{{ define "event_codec" }}{{ end -}}`))
	template.Must(tmpl.Parse(`{{ define "projections" }}{{ end -}}`))
	template.Must(tmpl.Parse(`{{ define "services" }}{{ end -}}`))

	outPkgPath, err := g.Generate(
		schema,
		root,
		gen.GeneratorOptions{
			PackageName:        "gencustomname",
			ExcludeProjections: false,
			TemplateTree:       tmpl,
		},
	)
	r.NoError(err)
	r.Equal(filepath.Join(root, "gencustomname"), outPkgPath)

	// Compile generated sources
	stdout, stderr, err := GoTool(root, "run", "./gencustomname")
	r.NoError(err, stderr)
	r.Equal("Hello, goesgen!", strings.TrimSpace(stdout))
}

func TestGenerate(t *testing.T) {
	r := require.New(t)

	root, files := Setup(t, ValidSetup)

	schema, err := gen.Parse(root, files["schema.yaml"])
	r.NoError(err)

	g := gen.NewGenerator()
	r.NotNil(g)

	outPkgPath, err := g.Generate(
		schema,
		root,
		gen.GeneratorOptions{
			PackageName:        "gencustomname",
			ExcludeProjections: false,
		},
	)
	r.NoError(err)
	r.Equal(filepath.Join(root, "gencustomname"), outPkgPath)

	// Check output files
	AssumeFilesExist(t, root,
		"schema.yaml",
		"cmd/main/main.go",
		"src.go",
		"sub/sub.go",
		"sub/subsub/subsub.go",
		"go.mod",
		"gencustomname/gencustomname.go",
	)

	// Compile generated sources
	_, stderr, err := GoTool(root, "build")
	r.NoError(err, stderr)
}

func AssumeFilesExist(t *testing.T, root string, expected ...string) {
	unexpected := []string{}
	require.NoError(t, filepath.Walk(
		root,
		func(ph string, info os.FileInfo, err error) error {
			require.NoError(t, err)
			if info.IsDir() {
				return nil
			}
			for _, x := range expected {
				p := filepath.Join(root, x)
				if p == ph {
					return nil
				}
			}
			unexpected = append(unexpected, ph[len(root)+1:])
			return nil
		},
	))
	require.Empty(t, unexpected,
		"%d unexpected file(s) in %s",
		len(unexpected), root,
	)
}
