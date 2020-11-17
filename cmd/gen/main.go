package main

import (
	"flag"
	"log"
	"text/template"

	"github.com/romshark/goesgen/gen"
)

func main() {
	flagSchemaPath := flag.String(
		"schema",
		"schema.yml",
		"schema YAML file path",
	)
	flagSourcePackagePath := flag.String(
		"src",
		".",
		"source package path",
	)
	flagOutputPath := flag.String(
		"out",
		"./",
		"generated package output path",
	)
	flagPackageName := flag.String(
		"pkgname",
		"",
		"generated package name",
	)
	flagExcludeProjections := flag.Bool(
		"exclude-projections",
		false,
		"disable projections generation",
	)
	flagUserTemplatesPath := flag.String(
		"templates",
		"",
		"path to user templates",
	)
	flag.Parse()

	s, err := gen.Parse(*flagSourcePackagePath, *flagSchemaPath)
	if err != nil {
		log.Fatalf("parsing schema/source: %s", err)
	}

	var t *template.Template
	if flagUserTemplatesPath != nil {
		t, err = template.ParseGlob(*flagUserTemplatesPath)
		if err != nil {
			log.Fatalf("parsing user templates: %s", err)
		}
	}

	outPackagePath, err := gen.NewGenerator().Generate(
		s, *flagOutputPath, gen.GeneratorOptions{
			PackageName:        *flagPackageName,
			TemplateTree:       t,
			ExcludeProjections: *flagExcludeProjections,
		},
	)
	if err != nil {
		log.Fatalf("generating: %s", err)
	}
	log.Printf("package successfully generated: %s", outPackagePath)
}
