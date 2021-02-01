package main

import (
	"errors"
	"fmt"
	goast "go/ast"
	goparser "go/parser"
	gotoken "go/token"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	oapi "github.com/getkin/kin-openapi/openapi3"
)

const (
	mimeJSON = "application/json"
)

var (
	ErrParserBadSpecs         = errors.New("bad specs")
	ErrParserNoSchema         = errors.New("no schema specified")
	ErrParserNoRootUrl        = errors.New("no root URL specified")
	ErrParserBadParamKind     = errors.New("bad parameter kind")
	ErrParserBadParamSchema   = errors.New("bad parameter schema")
	ErrParserBadRequestSchema = errors.New("bad request body schema")
)

type Parser struct {
	inpath   string
	srcpath  string
	specpath string

	rootURL    string
	modelPaths []string
	methods    map[string]*ServiceMethod

	// Used for template rendering, the 'true' AST.
	Services map[string]*ServiceInfo
}

type ServiceInfo struct {
	Filepath string
	Name     string
	Var      string
	Methods  map[string]*ServiceMethod
	Comment  string
}

type ServiceMethod struct {
	Receiver string
	Name     string
	Comment  string

	Path        string
	HttpMethod  string
	PathVars    []*PathVar
	Queries     []*Query
	RequestBody string
	Response    string
}

type PathVar struct {
	Name   string
	Type   string
	Field  string
	Binder string
}

type Query struct {
	Name  string
	Kind  string
	Field string
}

func NewParser() *Parser {
	return &Parser{
		Services: make(map[string]*ServiceInfo),
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
	serviceName := ServicePath(path).GetServiceName()
	serviceInfo := &ServiceInfo{
		Filepath: path,
		Name:     serviceName,
		Var:      serviceName,
		Methods:  make(map[string]*ServiceMethod),
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

	p.Services[serviceName] = serviceInfo
	return nil
}

func (p *Parser) collectModel(path string) {
	p.modelPaths = append(p.modelPaths, path)
}

func (p *Parser) parseYaml() error {
	swagger, err := oapi.NewSwaggerLoader().LoadSwaggerFromFile(p.specpath)
	if err != nil {
		return err
	}

	if err := p.parseRootURL(swagger.Servers); err != nil {
		return err
	}

	if err := p.parseServiceComments(swagger.Tags); err != nil {
		return err
	}

	for path, item := range swagger.Paths {
		if err := p.parseOperation(item.Get, path, http.MethodGet); err != nil {
			return err
		}
		if err := p.parseOperation(item.Post, path, http.MethodPost); err != nil {
			return err
		}
		if err := p.parseOperation(item.Put, path, http.MethodPut); err != nil {
			return err
		}
		if err := p.parseOperation(item.Delete, path, http.MethodDelete); err != nil {
			return err
		}
	}

	return nil
}

func (p *Parser) parseRootURL(servers []*oapi.Server) error {
	if len(servers) == 0 {
		return ErrParserNoRootUrl
	}

	rootURL, err := url.Parse(servers[0].URL)
	if err != nil {
		return err
	}

	p.rootURL = rootURL.Path
	return nil
}

func (p *Parser) parseServiceComments(tags oapi.Tags) error {
	for _, tag := range tags {
		serviceName := OapiTagToServiceName(tag.Name)
		service, ok := p.Services[serviceName]
		if !ok {
			return fmt.Errorf("%w: service %q not found in generated code",
				ErrParserBadSpecs, serviceName)
		}
		service.Comment = tag.Description
	}
	return nil
}

func (p *Parser) parseOperation(op *oapi.Operation, path, httpMethod string) error {
	if op == nil {
		return nil
	}

	id := strings.Title(op.OperationID)
	method, ok := p.methods[id]
	if !ok {
		return fmt.Errorf("%w: operation %q not found in generated code",
			ErrParserBadSpecs, id)
	}

	method.Comment = op.Summary
	method.Path = p.rootURL + OapiToGinPathParam(path)
	method.HttpMethod = httpMethod

	for _, param := range op.Parameters {
		if err := p.parseParam(method, param.Value); err != nil {
			return err
		}
	}

	if err := p.parseBody(method, op.RequestBody); err != nil {
		return err
	}

	if err := p.parseResponses(method, op.Responses); err != nil {
		return err
	}

	// TODO: Parse validation.

	return nil
}

func (p *Parser) parseParam(method *ServiceMethod, param *oapi.Parameter) error {
	m := method.Name
	name := param.Name
	schema := param.Schema

	if schema == nil {
		// TODO: Only parses JSON schema now.
		jsonSchema := param.Content.Get(mimeJSON)
		if jsonSchema == nil {
			return fmt.Errorf("%w: parameter %q of method %q", ErrParserNoSchema, name, m)
		}
		schema = jsonSchema.Schema
	}

	ty, err := OapiToGoType(schema)
	if err != nil {
		return fmt.Errorf("%w: cannot get Go type from param '%s/%s': %v",
			ErrParserBadParamSchema, m, name, err)
	}

	switch in := param.In; in {
	case "path":
		method.PathVars = append(method.PathVars, &PathVar{
			Name:   name,
			Type:   ty,
			Field:  strings.Title(name),
			Binder: "Param" + strings.Title(ty),
		})
	case "query":
		method.Queries = append(method.Queries, &Query{
			Name:  name,
			Kind:  ty,
			Field: strings.Title(name),
		})
	default:
		return fmt.Errorf("%w: %s", ErrParserBadParamKind, in)
	}

	return nil
}

func (p *Parser) parseBody(method *ServiceMethod, body *oapi.RequestBodyRef) error {
	if body == nil {
		return nil
	}

	m := method.Name

	// TODO
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
		return fmt.Errorf("%w: request body of method %q: %v", ErrParserBadRequestSchema, m, err)
	}
	if strings.HasPrefix(t, "Inline_object") {
		return fmt.Errorf("%w: request body %q of method %q: %v",
			ErrParserBadRequestSchema, t, m, ErrUtilUseRef)
	}

	method.RequestBody = t
	return nil
}

func (p *Parser) parseResponses(method *ServiceMethod, resps oapi.Responses) error {
	m := method.Name

	// TODO: Only 200 response is supported now, maybe we can use an interface
	// for all possible response schemas on 200, 400, etc. But hey, we don't
	// have sealed classes in Go :(
	resp := resps.Get(200)
	if resp == nil {
		return fmt.Errorf("%w: no 200 response given in method %q", ErrParserNoSchema, m)
	}

	// TODO
	jsonSchema := resp.Value.Content.Get(mimeJSON)
	if jsonSchema == nil {
		return fmt.Errorf("%w: response of method %q", ErrParserNoSchema, m)
	}

	ref := jsonSchema.Schema.Ref
	if ref == "" {
		return fmt.Errorf("%w: response of method %q", ErrUtilUseRef, m)
	}

	t, err := OapiRefToGoType(ref)
	if err != nil {
		return fmt.Errorf("%w: response schema of method %q: %v", ErrParserBadRequestSchema, m, err)
	}

	method.Response = t
	return nil
}
