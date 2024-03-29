package gen

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type Generator struct {
	tmpl *template.Template
}

//go:embed tmpl_generated.gtpl
var tmplGenerated string

//go:embed tmpl_events.gtpl
var tmplEvents string

//go:embed tmpl_event_codec.gtpl
var tmplEventCodec string

//go:embed tmpl_projections.gtpl
var tmplProjections string

//go:embed tmpl_services.gtpl
var tmplServices string

func NewGenerator() *Generator {
	t := template.Must(template.New("generated").Parse(tmplGenerated))
	template.Must(t.Parse(tmplEvents))
	template.Must(t.Parse(tmplEventCodec))
	template.Must(t.Parse(tmplProjections))
	template.Must(t.Parse(tmplServices))
	return &Generator{
		tmpl: t,
	}
}

func (g *Generator) Generate(
	schema *Schema,
	outputPath string,
	options GeneratorOptions,
) (outPackagePath string, err error) {
	if err := options.Prepare(); err != nil {
		return "", fmt.Errorf("preparing options: %w", err)
	}

	outPackagePath = filepath.Join(outputPath, options.PackageName)
	if err := os.MkdirAll(outPackagePath, 0777); err != nil {
		return "", fmt.Errorf("setting up %s: %w", outPackagePath, err)
	}
	if err := writeGoFile(
		nil,
		filepath.Join(outPackagePath, options.PackageName+".go"),
		g.tmpl,
		templateContext{
			Options: &options,
			Schema:  schema,
		},
	); err != nil {
		return "", err
	}
	return
}

type GeneratorOptions struct {
	PackageName        string
	ExcludeProjections bool
}

// Prepare validates the options and sets defaults for undefined values
func (o *GeneratorOptions) Prepare() error {
	if o.PackageName == "" {
		o.PackageName = "generated"
	} else {
		for _, c := range o.PackageName {
			if c < 'a' || c > 'z' {
				return fmt.Errorf("illegal package name (%q)", o.PackageName)
			}
		}
	}

	return nil
}

func writeGoFile(
	buf *bytes.Buffer,
	filePath string,
	tmpl *template.Template,
	data interface{},
) error {
	if buf == nil {
		buf = new(bytes.Buffer)
	}
	if err := tmpl.Execute(buf, data); err != nil {
		return fmt.Errorf("writing generated file: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("formatting file (%s): %w", filePath, err)
	}

	f, err := os.OpenFile(
		filePath,
		os.O_CREATE|os.O_TRUNC|os.O_WRONLY|os.O_SYNC,
		0644,
	)
	if err != nil {
		return fmt.Errorf("setting up %s: %w", filePath, err)
	}
	defer f.Close()
	if _, err := f.Write(formatted); err != nil {
		return fmt.Errorf("writing file (%s): %w", filePath, err)
	}
	return nil
}

type templateContext struct {
	Options *GeneratorOptions
	Schema  *Schema
}

func (templateContext) Capitalize(s string) string {
	return strings.Title(s)
}

func (templateContext) EventType(eventName string) string {
	return "Event" + eventName
}

func (templateContext) ProjectionType(projectionName string) string {
	return "Projection" + projectionName
}

func (templateContext) ProjectionStateConstant(
	projectionType, stateName string,
) string {
	return projectionType + "State" + stateName
}

func (templateContext) ServiceType(projectionName string) string {
	return "Service" + projectionName
}

func (templateContext) ImportAlias(p *SourcePackage) string {
	return "src" + strings.ReplaceAll(p.ID, ".", "")
}

func (c templateContext) TypeID(t *Type) string {
	return c.ImportAlias(t.Package) + "." + t.Name
}
