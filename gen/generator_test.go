package gen_test

import (
	"bytes"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"testing"

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

func TestGenerate(t *testing.T) {
	r := require.New(t)

	root, files := Setup(t, Files{
		"schema.yaml":      ValidSchemaSchemaYAML,
		"main.go":          ValidSchemaMainGO,
		"domain/domain.go": ValidSchemaDomainGO,
		"go.mod":           ValidSchemaGoMOD,
	})

	schema, err := gen.Parse(
		path.Join(root, "domain"),
		files["schema.yaml"],
	)
	r.NoError(err)

	g := gen.NewGenerator()
	r.NotNil(g)

	outPkgPath, err := g.Generate(
		schema,
		root,
		gen.GeneratorOptions{
			PackageName:        "customname",
			ExcludeProjections: false,
		},
	)
	r.NoError(err)
	r.Equal(path.Join(root, "customname"), outPkgPath)

	// Check output files
	AssumeFilesExist(t, root,
		"schema.yaml",
		"go.mod",
		"main.go",
		"domain/domain.go",
		"customname/customname.go",
	)

	// Compile generated sources
	cmd := exec.Command("go", "build")
	var errOut bytes.Buffer
	cmd.Stderr = &errOut
	cmd.Dir = root
	r.NoError(cmd.Run(), errOut.String())
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
				p := path.Join(root, x)
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
