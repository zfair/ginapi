package main

import (
	"errors"
	"fmt"
	goast "go/ast"
	goparser "go/parser"
	gotoken "go/token"
	"path/filepath"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

const (
	mimeJSON = "application/json"
)

var (
	ErrParserBadSpecs         = errors.New("bad specs")
	ErrParserNoSchema         = errors.New("no schema specified")
	ErrParserBadParamKind     = errors.New("bad parameter kind")
	ErrParserBadParamSchema   = errors.New("bad parameter schema")
	ErrParserBadRequestSchema = errors.New("bad request body schema")
)

type Parser struct {
	inpath   string
	srcpath  string
	specpath string

	services   map[string]*ServiceInfo
	methods    map[string]*ServiceMethod
	modelPaths []string
}

type ServiceInfo struct {
	Name    string
	Methods map[string]*ServiceMethod
}

type ServiceMethod struct {
	Name        string
	PathVars    []*PathVar
	Queries     []*Query
	RequestBody string
	Response    string
}

type PathVar struct {
	Var  string
	Type string
}

type Query struct {
	Key  string
	Kind string
}

func NewParser() *Parser {
	return &Parser{
		services: make(map[string]*ServiceInfo),
		methods:  make(map[string]*ServiceMethod),
	}
}

func (p *Parser) Parse() error {
	p.srcpath = filepath.Join(p.inpath, "go")
	p.specpath = filepath.Join(p.inpath, "api", "openapi.yaml")

	if err := p.parseGo(); err != nil {
		return err
	}

	if err := p.parseYaml(); err != nil {
		return err
	}

	return nil
}

func (p *Parser) parseGo() error {
	fset := gotoken.NewFileSet()
	pkgs, err := goparser.ParseDir(fset, p.srcpath, nil, goparser.ParseComments)
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		for path, file := range pkg.Files {
			if err := p.parseFile(path, file); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Parser) parseFile(path string, file *goast.File) error {
	filename := filepath.Base(path)

	if strings.HasPrefix(filename, "api_") {
		return p.parseMethodNames(filename, file)
	}
	if strings.HasPrefix(filename, "model_") {
		p.collectModel(path)
		return nil
	}
	if filename == "routers.go" {
		return p.parseRouters(file)
	}

	return nil
}

type ServicePath string

func (path ServicePath) GetServiceName() string {
	filename := filepath.Base(string(path))
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))

	parts := strings.Split(filename, "_")
	for i := 0; i < len(parts); i++ {
		parts[i] = strings.Title(parts[i])
	}

	parts = parts[1:]
	parts = append(parts, "Service")

	return strings.Join(parts, "")
}

func (p *Parser) parseMethodNames(path string, file *goast.File) error {
	serviceInfo := &ServiceInfo{
		Name:    ServicePath(path).GetServiceName(),
		Methods: make(map[string]*ServiceMethod),
	}

	for name, obj := range file.Scope.Objects {
		if obj.Kind != goast.Fun {
			continue
		}
		method := &ServiceMethod{
			Name: name,
		}
		serviceInfo.Methods[name] = method
		p.methods[name] = method
	}

	p.services[path] = serviceInfo
	return nil
}

func (p *Parser) collectModel(path string) {
	p.modelPaths = append(p.modelPaths, path)
}

func (p *Parser) parseRouters(file *goast.File) error {
	// TODO
	return nil
}

func (p *Parser) parseYaml() error {
	swagger, err := openapi3.NewSwaggerLoader().LoadSwaggerFromFile(p.specpath)
	if err != nil {
		return err
	}

	for _, pathItem := range swagger.Paths {
		if err := p.parseOperation(pathItem.Get); err != nil {
			return err
		}
		if err := p.parseOperation(pathItem.Post); err != nil {
			return err
		}
		if err := p.parseOperation(pathItem.Put); err != nil {
			return err
		}
		if err := p.parseOperation(pathItem.Delete); err != nil {
			return err
		}
	}

	return nil
}

func (p *Parser) parseOperation(op *openapi3.Operation) error {
	if op == nil {
		return nil
	}

	id := strings.Title(op.OperationID)
	method, ok := p.methods[id]
	if !ok {
		return fmt.Errorf("%w: operation %q not found in generated code",
			ErrParserBadSpecs, id)
	}

	for _, param := range op.Parameters {
		if err := p.parseParam(method, param.Value); err != nil {
			return err
		}
	}

	if err := p.parseBody(method, op.RequestBody); err != nil {
		return err
	}

	// TODO: Parse responses.

	// TODO: Parse validation.

	return nil
}

func (p *Parser) parseParam(method *ServiceMethod, param *openapi3.Parameter) error {
	name := param.Name
	schema := param.Schema

	if schema == nil {
		// NOTE: Only parses JSON schema.
		jsonSchema := param.Content.Get(mimeJSON)
		if jsonSchema == nil {
			return fmt.Errorf("%w: %s", ErrParserNoSchema, name)
		}
		schema = jsonSchema.Schema
	}

	ty, err := OapiToGoType(schema)
	if err != nil {
		return fmt.Errorf("%w: cannot get Go type from param '%s/%s': %v",
			ErrParserBadParamSchema, method.Name, name, err)
	}

	switch in := param.In; in {
	case "path":
		method.PathVars = append(method.PathVars, &PathVar{
			Var:  name,
			Type: ty,
		})
	case "query":
		method.Queries = append(method.Queries, &Query{
			Key:  name,
			Kind: ty,
		})
	default:
		return fmt.Errorf("%w: %s", ErrParserBadParamKind, in)
	}

	return nil
}

func (p *Parser) parseBody(method *ServiceMethod, body *openapi3.RequestBodyRef) error {
	if body == nil {
		return nil
	}

	m := method.Name

	jsonSchema := body.Value.Content.Get(mimeJSON)
	if jsonSchema == nil {
		return fmt.Errorf("%w: request body of method %q", ErrParserNoSchema, m)
	}
	ref := jsonSchema.Schema.Ref

	if ref == "" {
		return fmt.Errorf("%w: request body of method %q", ErrUtilUseRef, m)
	}

	t, err := OapiRefToGoType(ref)
	if err != nil {
		return fmt.Errorf("%w: method %q: %v", ErrParserBadRequestSchema, m, err)
	}
	if strings.HasPrefix(t, "Inline_object") {
		return fmt.Errorf("%w: request body %q of method %q: %v",
			ErrParserBadRequestSchema, t, m, ErrUtilUseRef)
	}

	method.RequestBody = t
	return nil
}
