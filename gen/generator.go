package gen

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path"
	"strings"
	"text/template"
)

type Generator struct {
	tmpl *template.Template
}

func NewGenerator() *Generator {
	t := template.Must(template.New("generated").Parse(TmplGenerated))
	template.Must(t.Parse(TmplEvents))
	template.Must(t.Parse(TmplEventCodec))
	template.Must(t.Parse(TmplProjections))
	template.Must(t.Parse(TmplServices))
	return &Generator{
		tmpl: t,
	}
}

func (g *Generator) Generate(
	schema *Schema,
	outputPath string,
	options GeneratorOptions,
) (outPackagePath string, err error) {
	outPackagePath = path.Join(outputPath, "generated")
	if err := os.MkdirAll(outPackagePath, 0777); err != nil {
		return "", fmt.Errorf("setting up %s: %w", outPackagePath, err)
	}
	if err := writeGoFile(
		nil,
		path.Join(outPackagePath, "generated.go"),
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
	ExcludeProjections bool
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
