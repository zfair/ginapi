package main

import (
	"errors"
	"fmt"
	goast "go/ast"
	goparser "go/parser"
	gotoken "go/token"
	"io/ioutil"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

var (
	ErrParserBadYaml = errors.New("bad yaml specs")
)

type Parser struct {
	inpath   string
	srcpath  string
	yamlpath string

	services   map[string]*ServiceInfo
	methods    map[string]*ServiceMethod
	modelPaths []string
}

type ServiceInfo struct {
	Name    string
	Methods map[string]*ServiceMethod
}

type ServiceMethod struct {
	Method      string
	PathVars    []string
	Queries     []*Query
	RequestBody string
	Response    string
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
	p.yamlpath = filepath.Join(p.inpath, "api", "openapi.yaml")

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
			Method: name,
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

type Specs map[interface{}]interface{}

func (specs Specs) GetAPIs() ([]Specs, error) {
	var ret []Specs
	rawPaths, ok := specs["paths"]

	if !ok {
		return nil, fmt.Errorf("%w: key \"paths\" not exists", ErrParserBadYaml)
	}
	paths, ok := rawPaths.(Specs)
	if !ok {
		return nil, fmt.Errorf("%w: key \"paths\" not a map", ErrParserBadYaml)
	}

	for _, rawPath := range paths {
		path, ok := rawPath.(Specs)
		if !ok {
			return nil, fmt.Errorf(
				"%w: key %q not a map",
				ErrParserBadYaml,
				rawPath,
			)
		}

		for _, rawInfo := range path {
			info, ok := rawInfo.(Specs)
			if !ok {
				return nil, fmt.Errorf(
					"%w: key %q not a map",
					ErrParserBadYaml,
					rawInfo,
				)
			}

			ret = append(ret, info)
		}
	}

	return ret, nil
}

func (p *Parser) parseYaml() error {
	file, err := ioutil.ReadFile(p.yamlpath)
	if err != nil {
		return err
	}

	specs := make(Specs)
	if err := yaml.Unmarshal(file, &specs); err != nil {
		return err
	}

	apis, err := specs.GetAPIs()
	if err != nil {
		return err
	}

	for _, api := range apis {
		if err := p.parseMethodInfo(api); err != nil {
			return err
		}
	}

	return nil
}

func (p *Parser) parseMethodInfo(api Specs) error {
	id, ok := api["operationId"]
	if !ok {
		return fmt.Errorf(
			"%w: key \"operationId\" not exists",
			ErrParserBadYaml,
		)
	}

	methodName := strings.Title(id.(string))
	method, ok := p.methods[methodName]
	if !ok {
		return fmt.Errorf(
			"%w: method %q not exists",
			ErrParserBadYaml,
			methodName,
		)
	}

	// TODO
	fmt.Println(method)
	return nil
}
