package gen

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"strings"

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
		Properties  map[PropertyName]TypeName       `yaml:"properties"`
		CreateOn    EventName                       `yaml:"createOn"`
		Transitions map[EventName][]ModelTransition `yaml:"transitions"`
	}
	ModelService struct {
		Projections []ProjectionName                  `yaml:"projections"`
		Methods     map[ServiceMethodName]ModelMethod `yaml:"methods"`
	}
	ModelMethod struct {
		Input  *TypeName         `yaml:"in"`
		Output *TypeName         `yaml:"out"`
		Type   ServiceMethodType `yaml:"type"`
		Emits  []EventName       `yaml:"emits"`
	}
	ModelEvent        map[PropertyName]TypeName
	ModelTransition   = string
	ServiceMethodName = string
	ServiceMethodType = string
	ServiceName       = string
	EventName         = string
	ProjectionName    = string
	ProjectionState   = string
	PropertyName      = string
	TypeName          = string
)

type (
	Schema struct {
		Raw             string
		Events          map[EventName]*Event
		Projections     map[ProjectionName]*Projection
		Services        map[ServiceName]*Service
		ReferencedTypes map[TypeName]*Type
		SourcePackage   string
		SourceModule    string
	}
	Type struct {
		Name           TypeName
		SourceLocation token.Position
		References     []interface{}
	}
	Projection struct {
		Schema       *Schema
		Name         ProjectionName
		States       map[ProjectionState]struct{}
		InitialState ProjectionState
		Properties   map[PropertyName]*Type
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
		Service *Service
		Name    ServiceMethodName
		Type    ServiceMethodType
		Input   *Type
		Output  *Type
		Emits   []*Event
	}
	Event struct {
		Schema     *Schema
		Name       string
		Properties map[PropertyName]*Type
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
	return ValidateTypeName(n)
}

func ValidateProjectionState(n ProjectionState) error {
	return ValidateTypeName(n)
}

func ValidateServiceName(n ServiceName) error {
	return ValidateTypeName(n)
}

func ValidateServiceMethodName(n ServiceMethodName) error {
	return ValidateTypeName(n)
}

func ValidateProjectionName(n ProjectionName) error {
	return ValidateTypeName(n)
}

func ValidatePropertyName(n PropertyName) error {
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

func ValidateTypeName(n string) error {
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
	s *Schema,
	m map[EventName]ModelEvent,
) error {
	if len(m) < 1 {
		return SemanticErr("missing event declarations")
	}
	s.Events = make(map[string]*Event, len(m))
	for n, e := range m {
		if err := ValidateEventName(n); err != nil {
			return syntaxErr("invalid event name (%q): %s", n, err)
		}
		v := &Event{
			Schema: s,
			Name:   n,
		}
		if err := parseEventProperties(s, v, e); err != nil {
			return err
		}
		s.Events[n] = v
	}
	return nil
}

func parseEventProperties(
	s *Schema,
	v *Event,
	m ModelEvent,
) error {
	v.Properties = make(map[string]*Type, len(m))
	for n, t := range m {
		if err := ValidatePropertyName(n); err != nil {
			return syntaxErr(
				"invalid property name (%q) in events.%s: %s",
				n, v.Name, err,
			)
		}
		if err := ValidateTypeName(t); err != nil {
			return syntaxErr(
				"invalid property type (%q) in events.%s: %s",
				t, v.Name, err,
			)
		}
		tp := registerReferencedType(s, t)
		v.Properties[n] = tp
		tp.References = append(tp.References, v)
	}
	return nil
}

func parseProjectionStates(
	p *Projection,
	m *ModelProjection,
) error {
	p.States = make(map[ProjectionState]struct{}, len(m.States))
	for i, s := range m.States {
		if err := ValidateProjectionState(s); err != nil {
			return syntaxErr(
				"invalid projection state (%q) "+
					"in projections.%s.states.%d: %s",
				s, p.Name, i, err,
			)
		}
		p.States[s] = struct{}{}
	}
	p.InitialState = m.States[0]
	return nil
}

func parseProjectionProperties(
	s *Schema,
	p *Projection,
	m *ModelProjection,
) error {
	p.Properties = make(map[PropertyName]*Type, len(m.Properties))
	for n, t := range m.Properties {
		if err := ValidatePropertyName(n); err != nil {
			return syntaxErr(
				"invalid property name (%q) in projections.%s.properties: %s",
				n, p.Name, err,
			)
		}
		if err := ValidateTypeName(t); err != nil {
			return syntaxErr(
				"invalid property type (%q) "+
					"in projections.%s.properties.%s: %s",
				t, p.Name, n, err,
			)
		}
		tp := registerReferencedType(s, t)
		p.Properties[n] = tp
		tp.References = append(tp.References, tp)
	}
	return nil
}

func parseProjectionCreateOn(
	s *Schema,
	p *Projection,
	m *ModelProjection,
) error {
	if _, ok := s.Events[m.CreateOn]; !ok {
		return semanticErr(
			"undefined event type %s in projections.%s.createOn",
			m.CreateOn, p.Name,
		)
	}
	e := s.Events[m.CreateOn]
	p.CreateOn = e
	e.References = append(e.References, p)
	return nil
}

func parseProjectionTransitions(
	s *Schema,
	p *Projection,
	m *ModelProjection,
) error {
	p.Transitions = make(map[*Event][]*Transition, len(m.Transitions))
	for e, t := range m.Transitions {
		for _, t := range t {
			v, ok := s.Events[e]
			if !ok {
				return semanticErr(
					"undefined event (%q) in projections.%s.transitions",
					e, p.Name,
				)
			}
			if err := parseTransition(p, v, t); err != nil {
				return err
			}
		}
	}
	return nil
}

func parseProjections(
	s *Schema,
	m map[ProjectionName]ModelProjection,
) error {
	s.Projections = make(map[string]*Projection, len(m))
	for n, pm := range m {
		if err := ValidateProjectionName(n); err != nil {
			return syntaxErr("invalid projection name (%q): %s", n, err)
		}
		p := &Projection{
			Name:   n,
			Schema: s,
		}
		if err := parseProjectionStates(p, &pm); err != nil {
			return err
		}
		if err := parseProjectionProperties(s, p, &pm); err != nil {
			return err
		}
		if err := parseProjectionCreateOn(s, p, &pm); err != nil {
			return err
		}
		if err := parseProjectionTransitions(s, p, &pm); err != nil {
			return err
		}

		for e := range p.Transitions {
			if e == p.CreateOn {
				return semanticErr(
					"event %s is used for both projections.%s.createOn "+
						"and projections.%s.transitions.%s",
					e.Name, p.Name, p.Name, e.Name,
				)
			}
		}

		s.Projections[p.Name] = p
	}
	return nil
}

func parseSchema(s *Schema, m *ModelSchema) error {
	if err := parseEvents(s, m.Events); err != nil {
		return err
	}
	if err := parseProjections(s, m.Projections); err != nil {
		return err
	}
	if err := parseServices(s, m.Services); err != nil {
		return err
	}

	for _, e := range s.Events {
		if len(e.References) < 1 {
			return semanticErr("unused event (%s)", e.Name)
		}
	}

	return nil
}

func parseTransition(
	p *Projection,
	e *Event,
	s string,
) error {
	f := strings.Fields(s)
	if len(f) != 3 || f[1] != "->" {
		return syntaxErr(
			"invalid expression format in projections.%s, "+
				"expected 'state -> state'",
			p.Name,
		)
	}
	from := ProjectionState(f[0])
	to := ProjectionState(f[2])

	// Check from-state
	if _, ok := p.States[from]; !ok {
		return semanticErr(
			"undefined from-state (%q) "+
				"in projections.%s.transitions.%s",
			from, p.Name, e.Name,
		)
	}

	// Check to-state
	if _, ok := p.States[to]; !ok {
		return semanticErr(
			"undefined to-state (%q) "+
				"in projections.%s.transitions.%s",
			to, p.Name, e.Name,
		)
	}

	// Check for redundant transitions
	if t, ok := p.Transitions[e]; ok {
		for _, t := range t {
			if t.On == e && t.From == from && t.To == to {
				return semanticErr(
					"duplicate transition (%s -> %s) "+
						"in projections.%s.transitions.%s",
					from, to, p.Name, e.Name,
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
	v *Service,
	s *Schema,
	m *ModelService,
) error {
	v.Projections = make([]*Projection, len(m.Projections))
	for i, pn := range m.Projections {
		p, ok := s.Projections[pn]
		if !ok {
			return semanticErr(
				"undefined projection (%q) in services.%s",
				pn, v.Name,
			)
		}
		v.Projections[i] = p
	}
	return nil
}

func parseServiceMethodInput(
	s *Schema,
	m *ServiceMethod,
	n *ServiceMethodName,
) error {
	if n == nil {
		return nil
	}
	if err := ValidateTypeName(*n); err != nil {
		return syntaxErr(
			"invalid type (%q) in services.%s.methods.%s.input: %s",
			*n, m.Service.Name, m.Name, err,
		)
	}
	m.Input = registerReferencedType(s, *n)
	m.Input.References = append(m.Input.References, m)
	return nil
}

func parseServiceMethodOutput(
	s *Schema,
	m *ServiceMethod,
	n *ServiceMethodName,
) error {
	if n == nil {
		return nil
	}
	if err := ValidateTypeName(*n); err != nil {
		return syntaxErr(
			"invalid type (%q) in services.%s.methods.%s.output: %s",
			*n, m.Service.Name, m.Name, err,
		)
	}
	m.Output = registerReferencedType(s, *n)
	m.Output.References = append(m.Output.References, m)
	return nil
}

func parseServiceMethods(
	v *Service,
	s *Schema,
	m *ModelService,
) error {
	if len(m.Methods) < 1 {
		return semanticErr("missing methods in services.%s", v.Name)
	}
	v.Methods = make(map[ServiceMethodName]*ServiceMethod, len(m.Methods))
	for name, model := range m.Methods {
		if err := ValidateServiceMethodName(name); err != nil {
			return syntaxErr(
				"invalid method name (%q) in services.%s: %s",
				name, v.Name, err,
			)
		}
		m := &ServiceMethod{
			Service: v,
			Name:    name,
		}
		if err := parseServiceMethodInput(s, m, model.Input); err != nil {
			return err
		}
		if err := parseServiceMethodOutput(s, m, model.Output); err != nil {
			return err
		}
		if err := parseServiceMethodEmits(s, m, model.Emits); err != nil {
			return err
		}
		if err := parseServiceMethodType(s, m, model.Type); err != nil {
			return err
		}
		v.Methods[name] = m
	}
	return nil
}

func parseServiceMethodType(
	s *Schema,
	m *ServiceMethod,
	t ServiceMethodType,
) error {
	illegalTypeErr := func() error {
		return syntaxErr(
			"illegal method type (%q) in services.%s.methods.%s.type",
			t, m.Service.Name, m.Name,
		)
	}
	if len(m.Emits) < 1 {
		switch t {
		case "", "readonly":
			m.Type = "readonly"
		case "append", "transaction":
			return semanticErr(
				"method type %s requires emits not to be empty "+
					"in services.%s.methods.%s",
				t, m.Service.Name, m.Name,
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
			return semanticErr(
				"method type can't be 'readonly' when emits is not empty "+
					"in services.%s.methods.%s",
				m.Service.Name, m.Name,
			)
		default:
			return illegalTypeErr()
		}
	}
	return nil
}

func parseServiceMethodEmits(
	s *Schema,
	m *ServiceMethod,
	emits []EventName,
) error {
	m.Emits = make([]*Event, len(emits))
	r := map[ServiceMethodName]struct{}{}
	for i, n := range emits {
		e, ok := s.Events[n]
		if !ok {
			return semanticErr(
				"undefined event (%q) in services.%s.methods.%s.emits.%d",
				n, m.Service.Name, m.Name, i,
			)
		}
		if _, ok := r[e.Name]; ok {
			return semanticErr(
				"duplicate event (%q) in services.%s.methods.%s.emits.%d",
				e.Name, m.Service.Name, m.Name, i,
			)
		}
		r[e.Name] = struct{}{}
		m.Emits[i] = e
		e.References = append(e.References, m)
	}
	return nil
}

func parseServices(
	s *Schema,
	m map[ServiceName]ModelService,
) error {
	if len(m) < 1 {
		return SemanticErr("missing service declarations")
	}
	s.Services = make(map[string]*Service, len(m))
	for n, v := range m {
		if err := ValidateServiceName(n); err != nil {
			return syntaxErr("invalid service name (%q): %s", n, err)
		}
		sv := &Service{
			Schema: s,
			Name:   n,
		}
		if err := parseServiceProjections(sv, s, &v); err != nil {
			return err
		}
		if err := parseServiceMethods(sv, s, &v); err != nil {
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

		s.Services[n] = sv
	}
	return nil
}

func registerReferencedType(s *Schema, typeName TypeName) *Type {
	if t, ok := s.ReferencedTypes[typeName]; ok {
		return t
	}
	t := &Type{
		Name: typeName,
	}
	s.ReferencedTypes[typeName] = t
	return t
}

func parseSources(sourcePackagePath string, s *Schema) error {
	tokenSet := token.NewFileSet()
	pkgs, err := parser.ParseDir(
		tokenSet,
		sourcePackagePath,
		nil,
		parser.AllErrors,
	)
	if err != nil {
		return semanticErr(
			"parsing source package (%s): %s",
			sourcePackagePath, err,
		)
	}

	for _, p := range pkgs {
		for _, fl := range p.Files {
			for _, o := range fl.Scope.Objects {
				if o.Kind != ast.Typ {
					continue
				}
				if t, ok := s.ReferencedTypes[o.Name]; ok {
					t.SourceLocation = tokenSet.Position(o.Pos())
				}
			}
		}
	}

	var undefinedTypes []string
	for n, t := range s.ReferencedTypes {
		if !t.SourceLocation.IsValid() {
			undefinedTypes = append(undefinedTypes, n)
		}
	}
	if undefinedTypes != nil {
		return semanticErr(
			"types (%s) undefined in source",
			strings.Join(undefinedTypes, ","),
		)
	}

	pkgInfo, err := packages.Load(&packages.Config{
		Dir: sourcePackagePath,
		Mode: packages.NeedName |
			packages.NeedDeps |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedModule,
	}, ".")
	if l := len(pkgInfo); l != 1 {
		return fmt.Errorf("unexpected number of source packages (%d)", l)
	}
	if err != nil {
		return fmt.Errorf(
			"parsing source package (%s): %w",
			sourcePackagePath, err,
		)
	}
	if len(pkgInfo[0].Errors) > 0 {
		return fmt.Errorf(
			"parsing source package (%s): %v",
			sourcePackagePath, pkgInfo[0].Errors,
		)
	}

	if pkgInfo[0].Module == nil {
		return fmt.Errorf(
			"source package (%s) is not a Go module (missing go.mod)",
			sourcePackagePath,
		)
	}
	s.SourceModule = pkgInfo[0].Module.Path
	s.SourcePackage = pkgInfo[0].Name

	return nil
}

func Parse(sourcePackagePath, schemaFilePath string) (*Schema, error) {
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

	d := yaml.NewDecoder(bytes.NewReader(flc))
	d.KnownFields(true)
	m := new(ModelSchema)
	if err := d.Decode(m); err != nil {
		return nil, syntaxErr(
			"parsing schema file (%s): %s",
			schemaFilePath, err,
		)
	}

	s := &Schema{
		Raw:             string(flc),
		ReferencedTypes: map[string]*Type{},
	}
	if err := parseSchema(s, m); err != nil {
		return nil, err
	}

	if err := parseSources(sourcePackagePath, s); err != nil {
		return nil, err
	}

	return s, nil
}

type SyntaxErr string

func (s SyntaxErr) Error() string { return "syntax error: " + string(s) }

func syntaxErr(format string, v ...interface{}) SyntaxErr {
	return SyntaxErr(fmt.Sprintf(format, v...))
}

type SemanticErr string

func (s SemanticErr) Error() string { return "semantic error: " + string(s) }

func semanticErr(format string, v ...interface{}) SemanticErr {
	return SemanticErr(fmt.Sprintf(format, v...))
}

func isLatinUpper(b byte) bool {
	return b >= 'A' && b <= 'Z'
}

func isLatinLower(b byte) bool {
	return b >= 'a' && b <= 'z'
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}
