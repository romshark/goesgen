package gen

import (
	"bytes"
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

func NewGenerator() *Generator {
	return &Generator{}
}

var builtinTemplates = map[string]string{
	"generated":   TmplGenerated,
	"events":      TmplEvents,
	"event_codec": TmplEventCodec,
	"projections": TmplProjections,
	"services":    TmplServices,
}

func (g *Generator) parseTemplates(t *template.Template) {
	g.tmpl = t
	for name, tmpl := range builtinTemplates {
		if g.tmpl == nil {
			g.tmpl = template.Must(template.New("generated").Parse(TmplGenerated))
		}
		// not defined by user -> use builtin
		if g.tmpl.Lookup(name) == nil {
			template.Must(g.tmpl.Parse(tmpl))
		}
	}
}

func (g *Generator) Generate(
	schema *Schema,
	outputPath string,
	options GeneratorOptions,
) (outPackagePath string, err error) {
	g.parseTemplates(options.TemplateTree)
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
	TemplateTree       *template.Template
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
		os.O_CREATE|os.O_WRONLY|os.O_SYNC,
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
