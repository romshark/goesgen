package gen_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"testing"

	"github.com/romshark/goesgen/gen"

	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	r := require.New(t)

	root, files := Setup(t, Files{
		"schema.yaml":      ValidSchemaSchemaYAML,
		"main.go":          ValidSchemaMainGO,
		"domain/domain.go": ValidSchemaDomainGO,
		"go.mod":           ValidSchemaGoMOD,
		"go.sum":           ValidSchemaGoSUM,
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
			ExcludeProjections: false,
		},
	)
	r.NoError(err)
	r.Equal(path.Join(root, "generated"), outPkgPath)

	// DEBUG
	// copyFile(
	// 	fmt.Sprintf("/Volumes/ramdisk/generated/generated.go"),
	// 	path.Join(root, "/generated/generated.go"),
	// )
	dp := path.Join("/Volumes/ramdisk/testgenroot/")
	if err := exec.Command("rm", "-rf", dp).Run(); err != nil {
		panic(fmt.Errorf("removing debug out dir: %w", err))
	}
	if err := os.MkdirAll(dp, 0777); err != nil {
		panic(err)
	}
	if err := exec.Command("cp", "-r", root, dp).Run(); err != nil {
		panic(fmt.Errorf("copying root path to debug out dir: %w", err))
	}

	// Check output files
	AssumeFilesExist(t, root,
		"schema.yaml",
		"go.mod",
		"go.sum",
		"main.go",
		"domain/domain.go",
		"generated/generated.go",
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
