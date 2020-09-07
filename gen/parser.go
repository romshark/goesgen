package gen

import (
	"bytes"
	"errors"
	"fmt"
	"go/token"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/tools/go/packages"
	"gopkg.in/yaml.v3"
)

type (
	ModelSchema struct {
		Events      map[EventName]ModelEvent           `yaml:"events"`
		Projections map[ProjectionName]ModelProjection `yaml:"projections"`
		Services    map[ServiceName]ModelService       `yaml:"services"`
	}
	ModelProjection struct {
		States      []ProjectionState               `yaml:"states"`
		Properties  ModelProperties                 `yaml:"properties"`
		CreateOn    EventName                       `yaml:"createOn"`
		Transitions map[EventName][]ModelTransition `yaml:"transitions"`
	}
	ModelService struct {
		Projections []ProjectionName    `yaml:"projections"`
		Methods     ModelServiceMethods `yaml:"methods"`
	}
	ModelServiceMethod struct {
		Pos          int
		CommentLines []string
		Input        *TypeID           `yaml:"in"`
		Output       *TypeID           `yaml:"out"`
		Type         ServiceMethodType `yaml:"type"`
		Emits        []EventName       `yaml:"emits"`
	}
	ModelEvent      = ModelProperties
	ModelProperties struct {
		events map[PropertyName]ModelProperty
	}
	ModelProperty struct {
		Pos          int
		TypeID       TypeID
		CommentLines []string
	}
	ModelServiceMethods struct {
		methods map[ServiceMethodName]ModelServiceMethod
	}
	ModelTransition   = string
	ServiceMethodName = string
	ServiceMethodType = string
	ServiceName       = string
	EventName         = string
	ProjectionName    = string
	ProjectionState   = string
	PropertyName      = string
	TypeID            = string
)

func (m *ModelProperties) UnmarshalYAML(v *yaml.Node) error {
	if len(v.Content)%2 != 0 {
		return fmt.Errorf(
			"unexpected number of nodes (%d) in event properties",
			len(v.Content),
		)
	}
	m.events = make(map[PropertyName]ModelProperty, len(v.Content)/2)

	for i, pos := 0, 0; i < len(v.Content); i, pos = i+2, pos+1 {
		nameNode := v.Content[i]
		typeNode := v.Content[i+1]

		commentLines := ParseComment(nameNode.HeadComment)
		if len(commentLines) > 0 {
			n := strings.Title(nameNode.Value)
			commentLines[0] = strings.Title(commentLines[0])
			if strings.Fields(commentLines[0])[0] != n {
				return fmt.Errorf(
					"illegal event property comment at %d:%d, "+
						"must begin with %q",
					nameNode.Line, nameNode.Column, n+"...",
				)
			}
		}

		m.events[nameNode.Value] = ModelProperty{
			Pos:          pos,
			TypeID:       typeNode.Value,
			CommentLines: commentLines,
		}
	}
	return nil
}

func (m *ModelServiceMethods) UnmarshalYAML(v *yaml.Node) error {
	m.methods = make(
		map[ServiceMethodName]ModelServiceMethod,
		len(v.Content)/2,
	)

	for i, pos := 0, 0; i < len(v.Content); i, pos = i+2, pos+1 {
		nameNode := v.Content[i]
		methodNode := v.Content[i+1]

		commentLines := ParseComment(nameNode.HeadComment)
		if len(commentLines) > 0 {
			n := strings.Title(nameNode.Value)
			commentLines[0] = strings.Title(commentLines[0])
			if strings.Fields(commentLines[0])[0] != n {
				return fmt.Errorf(
					"illegal service method comment at %d:%d, "+
						"must begin with %q",
					nameNode.Line, nameNode.Column, n+"...",
				)
			}
		}

		v := ModelServiceMethod{
			Pos:          pos,
			CommentLines: commentLines,
		}

		for i := 0; i < len(methodNode.Content); i++ {
			switch c := methodNode.Content[i]; c.Value {
			case "in":
				i++
				v.Input = &methodNode.Content[i].Value
			case "out":
				i++
				v.Output = &methodNode.Content[i].Value
			case "type":
				i++
				v.Type = methodNode.Content[i].Value
			case "emits":
				i++
				v.Emits = make(
					[]EventName,
					len(methodNode.Content[i].Content),
				)
				for i, c := range methodNode.Content[i].Content {
					v.Emits[i] = c.Value
				}
			default:
				return fmt.Errorf(
					`unexpected field %q (expected either of %q) at %d:%d`,
					c.Value, "in, out, type, emits", c.Line, c.Column,
				)
			}
		}

		m.methods[nameNode.Value] = v
	}
	return nil
}

type (
	SourcePackageName = string
	SourcePackageID   = string
	SourcePackage     struct {
		Path       string // Absolute directory path
		ImportPath string // Import path relative to the module
		Name       SourcePackageName
		ID         SourcePackageID // dot-separated unique identifier
		Types      map[TypeID]*Type
	}
	Schema struct {
		Raw            string
		Events         map[EventName]*Event
		Projections    map[ProjectionName]*Projection
		Services       map[ServiceName]*Service
		SourcePackages map[SourcePackageID]*SourcePackage
		SourcePackage  *SourcePackage
		SourceModule   string
	}
	Type struct {
		ID             string
		Name           string
		Package        *SourcePackage
		SourceLocation token.Position
		References     []interface{}
	}
	Projection struct {
		Schema       *Schema
		Name         ProjectionName
		States       map[ProjectionState]struct{}
		InitialState ProjectionState
		Properties   []*Property
		CreateOn     *Event
		Transitions  map[*Event][]*Transition
	}
	Service struct {
		Schema        *Schema
		Name          ServiceName
		Projections   []*Projection
		Methods       map[ServiceMethodName]*ServiceMethod
		Subscriptions map[EventName]*Event
	}
	ServiceMethod struct {
		Service      *Service
		Name         ServiceMethodName
		Type         ServiceMethodType
		Input        *Type
		Output       *Type
		Emits        []*Event
		CommentLines []string
	}
	Property struct {
		Position     int
		Name         PropertyName
		Type         *Type
		CommentLines []string
	}
	Event struct {
		Schema     *Schema
		Name       string
		Properties []*Property
		References []interface{}
	}
	Transition struct {
		Projection *Projection
		On         *Event
		From       ProjectionState
		To         ProjectionState
	}
)

func ValidateEventName(n EventName) error {
	return ValidatePascalCase(n)
}

func ValidateProjectionState(n ProjectionState) error {
	return ValidatePascalCase(n)
}

func ValidateServiceName(n ServiceName) error {
	return ValidatePascalCase(n)
}

func ValidateServiceMethodName(n ServiceMethodName) error {
	return ValidatePascalCase(n)
}

func ValidateProjectionName(n ProjectionName) error {
	return ValidatePascalCase(n)
}

func ValidatePropertyName(n PropertyName) error {
	return ValidateCamelCase(n)
}

func ValidatePascalCase(n string) error {
	if len(n) < 1 {
		return ErrEmpty
	}
	if !isLatinUpper(n[0]) {
		return ErrMustBeginWithUpperLatin
	}
	for i := range n[1:] {
		if isLatinLower(n[i]) ||
			isLatinUpper(n[i]) ||
			isDigit(n[i]) ||
			n[i] == '_' {
			continue
		}
		return ErrContainsIllegalChars
	}
	return nil
}

func ValidateCamelCase(n string) error {
	if len(n) < 1 {
		return ErrEmpty
	}
	if !isLatinLower(n[0]) {
		return ErrMustBeginWithLowerLatin
	}
	for i := range n[1:] {
		if isLatinLower(n[i]) ||
			isLatinUpper(n[i]) ||
			isDigit(n[i]) ||
			n[i] == '_' {
			continue
		}
		return ErrContainsIllegalChars
	}
	return nil
}

func ParseTypeID(t TypeID) (
	typeName string,
	packagePath []string,
	err error,
) {
	p := strings.Split(t, ".")
	if l := len(p); l < 2 {
		if err = ValidatePascalCase(t); err != nil {
			return
		}
		return t, nil, nil
	}
	return p[len(p)-1], p[:len(p)-1], nil
}

var (
	ErrMustBeginWithLowerLatin = errors.New(
		"must begin with a lower case latin character",
	)
	ErrMustBeginWithUpperLatin = errors.New(
		"must begin with an upper case latin character",
	)
	ErrContainsIllegalChars = errors.New(
		"contains illegal characters",
	)
	ErrEmpty = errors.New("empty")
)

func parseEvents(
	ctx context,
	m map[EventName]ModelEvent,
) error {
	if len(m) < 1 {
		return ctx.semanticErr("missing event declarations")
	}
	ctx.schema.Events = make(map[string]*Event, len(m))
	for n, e := range m {
		if err := ValidateEventName(n); err != nil {
			return ctx.syntaxErr("invalid event name (%q): %s", n, err)
		}
		v := &Event{
			Schema: ctx.schema,
			Name:   n,
		}
		if err := parseEventProperties(
			ctx.Subcontext("properties"), v, e,
		); err != nil {
			return err
		}
		ctx.schema.Events[n] = v
	}
	return nil
}

func parseEventProperties(
	ctx context,
	v *Event,
	m ModelEvent,
) error {
	v.Properties = make([]*Property, len(m.events))
	for n, t := range m.events {
		if err := ValidatePropertyName(n); err != nil {
			return ctx.syntaxErr("invalid property name (%q): %s", n, err)
		}
		tp, err := registerReferencedType(ctx.Subcontext(n), t.TypeID)
		if err != nil {
			return ctx.syntaxErr("invalid type identifier (%q): %s", t, err)
		}
		v.Properties[t.Pos] = &Property{
			Position:     t.Pos,
			Name:         n,
			Type:         tp,
			CommentLines: t.CommentLines,
		}
		tp.References = append(tp.References, v)
	}
	return nil
}

func parseProjectionStates(
	ctx context,
	p *Projection,
	m *ModelProjection,
) error {
	p.States = make(map[ProjectionState]struct{}, len(m.States))
	for i, s := range m.States {
		if err := ValidateProjectionState(s); err != nil {
			return ctx.Subcontext(strconv.Itoa(i)).
				syntaxErr("invalid projection state (%q): %s", s, err)
		}
		p.States[s] = struct{}{}
	}
	p.InitialState = m.States[0]
	return nil
}

func parseProjectionProperties(
	ctx context,
	p *Projection,
	m *ModelProjection,
) error {
	p.Properties = make([]*Property, len(m.Properties.events))
	for n, t := range m.Properties.events {
		if err := ValidatePropertyName(n); err != nil {
			return ctx.syntaxErr("invalid property name (%q): %s", n, err)
		}
		tp, err := registerReferencedType(ctx.Subcontext(n), t.TypeID)
		if err != nil {
			return ctx.Subcontext(n).
				syntaxErr("invalid type identifier (%q): %s", t, err)
		}
		p.Properties[t.Pos] = &Property{
			Position:     t.Pos,
			Name:         n,
			Type:         tp,
			CommentLines: t.CommentLines,
		}
		tp.References = append(tp.References, tp)
	}
	return nil
}

func parseProjectionCreateOn(
	ctx context,
	p *Projection,
	m *ModelProjection,
) error {
	if _, ok := ctx.schema.Events[m.CreateOn]; !ok {
		return ctx.semanticErr("undefined event type %s", m.CreateOn)
	}
	e := ctx.schema.Events[m.CreateOn]
	p.CreateOn = e
	e.References = append(e.References, p)
	return nil
}

func parseProjectionTransitions(
	ctx context,
	p *Projection,
	m *ModelProjection,
) error {
	p.Transitions = make(map[*Event][]*Transition, len(m.Transitions))
	for e, t := range m.Transitions {
		for i, t := range t {
			v, ok := ctx.schema.Events[e]
			if !ok {
				return ctx.semanticErr("undefined event (%q)", e)
			}
			if err := parseTransition(
				ctx.Subcontext(strconv.Itoa(i)), p, v, t,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func parseProjections(
	ctx context,
	m map[ProjectionName]ModelProjection,
) error {
	ctx.schema.Projections = make(map[string]*Projection, len(m))
	for n, pm := range m {
		ctx := ctx.Subcontext(n)

		if err := ValidateProjectionName(n); err != nil {
			return ctx.syntaxErr("invalid projection name (%q): %s", n, err)
		}
		p := &Projection{
			Name:   n,
			Schema: ctx.schema,
		}
		if err := parseProjectionStates(
			ctx.Subcontext("states"), p, &pm,
		); err != nil {
			return err
		}
		if err := parseProjectionProperties(
			ctx.Subcontext("properties"), p, &pm,
		); err != nil {
			return err
		}
		if err := parseProjectionCreateOn(
			ctx.Subcontext("createOn"), p, &pm,
		); err != nil {
			return err
		}
		if err := parseProjectionTransitions(
			ctx.Subcontext("transitions"), p, &pm,
		); err != nil {
			return err
		}

		for e := range p.Transitions {
			if e == p.CreateOn {
				return ctx.semanticErr(
					"event %s is used for both %s and %s",
					e.Name,
					ctx.Subcontext("createOn").path,
					ctx.Subcontext("transitions", e.Name).path,
				)
			}
		}

		ctx.schema.Projections[p.Name] = p
	}
	return nil
}

func parseSchema(
	ctx context,
	m *ModelSchema,
) error {
	if err := parseEvents(
		ctx.Subcontext("events"),
		m.Events,
	); err != nil {
		return err
	}
	if err := parseProjections(
		ctx.Subcontext("projections"),
		m.Projections,
	); err != nil {
		return err
	}
	if err := parseServices(
		ctx.Subcontext("services"),
		m.Services,
	); err != nil {
		return err
	}

	for _, e := range ctx.schema.Events {
		if len(e.References) < 1 {
			return ctx.semanticErr("unused event (%s)", e.Name)
		}
	}

	return nil
}

func parseTransition(
	ctx context,
	p *Projection,
	e *Event,
	s string,
) error {
	f := strings.Fields(s)
	if len(f) != 3 || f[1] != "->" {
		return ctx.syntaxErr(
			"invalid expression format, expected 'state -> state'",
		)
	}
	from := ProjectionState(f[0])
	to := ProjectionState(f[2])

	// Check from-state
	if _, ok := p.States[from]; !ok {
		return ctx.semanticErr("undefined from-state (%q)", from)
	}

	// Check to-state
	if _, ok := p.States[to]; !ok {
		return ctx.semanticErr("undefined to-state (%q)", to)
	}

	// Check for redundant transitions
	if t, ok := p.Transitions[e]; ok {
		for _, t := range t {
			if t.On == e && t.From == from && t.To == to {
				return ctx.semanticErr(
					"duplicate transition (%s -> %s)",
					from, to,
				)
			}
		}
	}

	t := &Transition{
		Projection: p,
		On:         e,
		From:       from,
		To:         to,
	}
	p.Transitions[e] = append(p.Transitions[e], t)
	e.References = append(e.References, t)
	return nil
}

func parseServiceProjections(
	ctx context,
	v *Service,
	m *ModelService,
) error {
	v.Projections = make([]*Projection, len(m.Projections))
	for i, pn := range m.Projections {
		p, ok := ctx.schema.Projections[pn]
		if !ok {
			return ctx.semanticErr("undefined projection (%q)", pn)
		}
		v.Projections[i] = p
	}
	return nil
}

func parseServiceMethodInput(
	ctx context,
	m *ServiceMethod,
	n *ServiceMethodName,
) error {
	if n == nil {
		return nil
	}
	var err error
	m.Input, err = registerReferencedType(ctx, *n)
	if err != nil {
		return err
	}
	m.Input.References = append(m.Input.References, m)
	return nil
}

func parseServiceMethodOutput(
	ctx context,
	m *ServiceMethod,
	n *ServiceMethodName,
) error {
	if n == nil {
		return nil
	}
	var err error
	m.Output, err = registerReferencedType(ctx, *n)
	if err != nil {
		return err
	}
	m.Output.References = append(m.Output.References, m)
	return nil
}

func parseServiceMethods(
	ctx context,
	v *Service,
	m *ModelService,
) error {
	if len(m.Methods.methods) < 1 {
		return ctx.semanticErr("missing methods")
	}
	v.Methods = make(
		map[ServiceMethodName]*ServiceMethod,
		len(m.Methods.methods),
	)
	for name, model := range m.Methods.methods {
		if err := ValidateServiceMethodName(name); err != nil {
			return ctx.syntaxErr(
				"invalid method name (%q): %s",
				name, err,
			)
		}
		m := &ServiceMethod{
			Service:      v,
			Name:         name,
			CommentLines: model.CommentLines,
		}
		if err := parseServiceMethodInput(
			ctx.Subcontext("input"),
			m,
			model.Input,
		); err != nil {
			return err
		}
		if err := parseServiceMethodOutput(
			ctx.Subcontext("output"),
			m,
			(*string)(model.Output),
		); err != nil {
			return err
		}
		if err := parseServiceMethodEmits(
			ctx.Subcontext("emits"),
			m,
			model.Emits,
		); err != nil {
			return err
		}
		if err := parseServiceMethodType(
			ctx.Subcontext("type"),
			m,
			model.Type,
		); err != nil {
			return err
		}
		v.Methods[name] = m
	}
	return nil
}

func parseServiceMethodType(
	ctx context,
	m *ServiceMethod,
	t ServiceMethodType,
) error {
	illegalTypeErr := func() error {
		return ctx.syntaxErr("illegal method type (%q)", t)
	}
	if len(m.Emits) < 1 {
		switch t {
		case "", "readonly":
			m.Type = "readonly"
		case "append", "transaction":
			return ctx.semanticErr(
				"method type %s requires emits not to be empty", t,
			)
		default:
			return illegalTypeErr()
		}
	} else {
		switch t {
		case "", "transaction":
			m.Type = "transaction"
		case "append":
			m.Type = "append"
		case "readonly":
			return ctx.semanticErr(
				"method type can't be 'readonly' when emits is not empty",
			)
		default:
			return illegalTypeErr()
		}
	}
	return nil
}

func parseServiceMethodEmits(
	ctx context,
	m *ServiceMethod,
	emits []EventName,
) error {
	m.Emits = make([]*Event, len(emits))
	r := map[ServiceMethodName]struct{}{}
	for i, n := range emits {
		e, ok := ctx.schema.Events[n]
		if !ok {
			return ctx.Subcontext(strconv.Itoa(i)).
				semanticErr("undefined event (%q)", n)
		}
		if _, ok := r[e.Name]; ok {
			return ctx.Subcontext(strconv.Itoa(i)).
				semanticErr("duplicate event (%q)", e.Name)
		}
		r[e.Name] = struct{}{}
		m.Emits[i] = e
		e.References = append(e.References, m)
	}
	return nil
}

func parseServices(
	ctx context,
	m map[ServiceName]ModelService,
) error {
	if len(m) < 1 {
		return ctx.semanticErr("missing service declarations")
	}
	ctx.schema.Services = make(map[string]*Service, len(m))
	for n, v := range m {
		ctx := ctx.Subcontext(n)

		if err := ValidateServiceName(n); err != nil {
			return ctx.syntaxErr("invalid service name (%q): %s", n, err)
		}
		sv := &Service{
			Schema: ctx.schema,
			Name:   n,
		}
		if err := parseServiceProjections(
			ctx.Subcontext("projections"), sv, &v,
		); err != nil {
			return err
		}
		if err := parseServiceMethods(
			ctx.Subcontext("methods"), sv, &v,
		); err != nil {
			return err
		}

		// Determine subscriptions
		sv.Subscriptions = make(map[EventName]*Event, len(sv.Projections))
		for _, p := range sv.Projections {
			sv.Subscriptions[p.CreateOn.Name] = p.CreateOn
			for e := range p.Transitions {
				sv.Subscriptions[e.Name] = e
			}
		}

		ctx.schema.Services[n] = sv
	}
	return nil
}

// registerReferencedType registers a new referenced type
// and returns it. If the given type is already registered
// it is returned instead.
func registerReferencedType(
	ctx context,
	tid TypeID,
) (*Type, error) {
	typeName, importPath, err := ParseTypeID(tid)
	if err != nil {
		return nil, ctx.syntaxErr(
			"invalid type identifier (%q): %s",
			tid, err,
		)
	}

	pkgName := ctx.schema.SourcePackage.Name
	pkgID := pkgName
	if len(importPath) > 0 {
		pkgName = importPath[len(importPath)-1]
		pkgID = ctx.schema.SourcePackage.ID +
			"." +
			strings.Join(importPath, ".")
	}

	pkg, ok := ctx.schema.SourcePackages[pkgID]
	if !ok {
		pkg = &SourcePackage{
			ID: pkgID,
			Path: path.Join(
				ctx.schema.SourcePackage.Path,
				path.Join(importPath...),
			),
			ImportPath: path.Join(importPath...),
			Name:       pkgName,
			Types:      make(map[TypeID]*Type, 1),
		}
		ctx.schema.SourcePackages[pkgID] = pkg
	}

	id := pkgID + "." + typeName
	if t, ok := pkg.Types[id]; ok {
		return t, nil
	}
	t := &Type{
		ID:      id,
		Name:    typeName,
		Package: pkg,
	}
	pkg.Types[id] = t
	return t, nil
}

func parseSources(
	ctx context,
	sourcePackagePath string,
) error {
	type Pkg struct {
		Pkg  *packages.Package
		Fset *token.FileSet
	}
	pis := map[SourcePackageID]Pkg{}

	loadPackage := func(p *SourcePackage) error {
		fset := token.NewFileSet()
		pkgInfo, err := packages.Load(&packages.Config{
			Dir:  p.Path,
			Fset: fset,
			Mode: packages.NeedName |
				packages.NeedDeps |
				packages.NeedTypes |
				packages.NeedSyntax |
				packages.NeedModule,
		}, ".")
		if len(pkgInfo) != 1 {
			return fmt.Errorf("package (%q) not found", p.Path)
		}
		if err != nil {
			return fmt.Errorf(
				"parsing source package (%s): %w",
				sourcePackagePath, err,
			)
		}
		pi := pkgInfo[0]
		if len(pi.Errors) > 0 {
			return fmt.Errorf(
				"parsing source package (%s): %v",
				sourcePackagePath, pi.Errors,
			)
		}

		if pi.Module == nil {
			return fmt.Errorf(
				"source package (%s) is not a Go module (missing go.mod)",
				sourcePackagePath,
			)
		}

		if ctx.schema.SourcePackage == p {
			// Main source package
			ctx.schema.SourceModule = pi.Module.Path
			ctx.schema.SourcePackage.ImportPath = path.Join(
				pi.Module.Path,
				ctx.schema.SourcePackage.Name,
			)
		} else {
			p.ImportPath = path.Join(
				ctx.schema.SourceModule,
				ctx.schema.SourcePackage.Name,
				p.ImportPath,
			)
			if ctx.schema.SourceModule != pi.Module.Path {
				// Subpackage
				return ctx.semanticErr(
					"package %s (%s) isn't part of the source module (%s)",
					p.ID, p.Path, ctx.schema.SourceModule,
				)
			}
		}

		pis[p.ID] = Pkg{pi, fset}

		return nil
	}

	if err := loadPackage(
		ctx.schema.SourcePackages[ctx.schema.SourcePackage.ID],
	); err != nil {
		return err
	}
	for _, p := range ctx.schema.SourcePackages {
		if p == ctx.schema.SourcePackage {
			// Ignore the main package
			continue
		}
		if err := loadPackage(p); err != nil {
			return err
		}
	}

	// Determine type source locations
	for _, p := range ctx.schema.SourcePackages {
		pk := pis[p.ID]
		s := pk.Pkg.Types.Scope()
		for _, t := range p.Types {
			if typ := s.Lookup(t.Name); typ != nil {
				t.SourceLocation = pk.Fset.Position(typ.Pos())
			}
		}
	}

	// Check for undefined types
	var undefinedTypes []string
	for _, p := range ctx.schema.SourcePackages {
		for _, t := range p.Types {
			if !t.SourceLocation.IsValid() {
				undefinedTypes = append(undefinedTypes, string(t.ID))
			}
		}
	}
	if undefinedTypes != nil {
		return ctx.semanticErr(
			"types (%s) undefined",
			strings.Join(undefinedTypes, ","),
		)
	}

	return nil
}

func Parse(
	sourcePackagePath,
	schemaFilePath string,
) (*Schema, error) {
	fl, err := os.OpenFile(
		schemaFilePath,
		os.O_RDONLY,
		0644,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"reading schema file (%s): %q",
			schemaFilePath, err,
		)
	}

	flc, err := ioutil.ReadAll(fl)
	if err != nil {
		return nil, fmt.Errorf(
			"reading schema file (%s) into memory: %q",
			schemaFilePath, err,
		)
	}

	s := &Schema{
		Raw: string(flc),
		SourcePackage: &SourcePackage{
			Types: map[TypeID]*Type{},
		},
	}
	ctx := context{schema: s}

	{ // Determine package path and name
		a, err := filepath.Abs(sourcePackagePath)
		if err != nil {
			return nil, fmt.Errorf(
				"determining absolute source package path: %w",
				err,
			)
		}
		p := s.SourcePackage
		p.Path = a

		_, p.Name = filepath.Split(a)
		p.ID = p.Name
	}

	s.SourcePackages = map[SourcePackageID]*SourcePackage{
		s.SourcePackage.ID: s.SourcePackage,
	}

	d := yaml.NewDecoder(bytes.NewReader(flc))
	d.KnownFields(true)
	m := new(ModelSchema)
	if err := d.Decode(m); err != nil {
		return nil, ctx.syntaxErr(
			"parsing schema file (%s): %s",
			schemaFilePath, err,
		)
	}

	if err := parseSchema(ctx, m); err != nil {
		return nil, err
	}

	if err := parseSources(ctx, sourcePackagePath); err != nil {
		return nil, err
	}

	return s, nil
}

func ParseComment(s string) []string {
	f := strings.FieldsFunc(
		strings.TrimSpace(s),
		func(r rune) bool { return r == '\n' },
	)
FLOOP:
	for i := range f {
		s := f[i]
		for j, c := range s {
			if c != '#' && !unicode.IsSpace(c) {
				f[i] = strings.TrimSpace(s[j:])
				continue FLOOP
			}
		}
		f[i] = ""
	}
	return f
}

type SyntaxErr string

func (s SyntaxErr) Error() string { return "syntax error: " + string(s) }

type SemanticErr string

func (s SemanticErr) Error() string { return "semantic error: " + string(s) }

func isLatinUpper(b byte) bool {
	return b >= 'A' && b <= 'Z'
}

func isLatinLower(b byte) bool {
	return b >= 'a' && b <= 'z'
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

type context struct {
	schema *Schema
	path   string
}

func (c context) Subcontext(pathElements ...string) context {
	newPath := strings.Join(pathElements, ".")
	if c.path != "" {
		newPath = c.path + "." + newPath
	}
	return context{
		schema: c.schema,
		path:   newPath,
	}
}

func (c context) syntaxErr(
	format string,
	v ...interface{},
) SyntaxErr {
	msg := fmt.Sprintf(format, v...)
	if c.path != "" {
		msg = c.path + ": " + msg
	}
	return SyntaxErr(msg)
}

func (c context) semanticErr(
	format string,
	v ...interface{},
) SemanticErr {
	msg := fmt.Sprintf(format, v...)
	if c.path != "" {
		msg = c.path + ": " + msg
	}
	return SemanticErr(msg)
}
