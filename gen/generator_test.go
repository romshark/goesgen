package gen_test

import (
	"bytes"
	"os"
	"os/exec"
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

	root, files := Setup(t, ValidSetup)

	schema, err := gen.Parse(
		filepath.Join(root, "src"),
		files["schema.yaml"],
	)
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
		"main.go",
		"src/src.go",
		"src/sub/sub.go",
		"src/sub/subsub/subsub.go",
		"go.mod",
		"gencustomname/gencustomname.go",
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
