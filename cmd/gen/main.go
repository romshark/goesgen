package main

import (
	"flag"
	"log"

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
	flagExcludeProjections := flag.Bool(
		"exclude-projections",
		false,
		"disable projections generation",
	)
	flag.Parse()

	s, err := gen.Parse(*flagSourcePackagePath, *flagSchemaPath)
	if err != nil {
		log.Fatalf("parsing schema/source: %s", err)
	}

	outPackagePath, err := gen.NewGenerator().Generate(
		s, *flagOutputPath, gen.GeneratorOptions{
			ExcludeProjections: *flagExcludeProjections,
		},
	)
	if err != nil {
		log.Fatalf("generating: %s", err)
	}
	log.Printf("package successfully generated: %s", outPackagePath)
}
