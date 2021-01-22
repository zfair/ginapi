package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"
)

const (
	CodegenModeServer = "server"
	CodegenModeClient = "client"

	filePerm = 0644

	tmplFileHeader = `// Generated by ginapi. DO NOT EDIT.
package ginapi
`

	commonFileTmpl = tmplFileHeader + `

import (
	"github.com/gin-gonic/gin"
)

type ginRegistry struct {
	HttpMethod string
	URL string
	PreMain []gin.HandlerFunc
	Main gin.HandlerFunc
	PostMain []gin.HandlerFunc
}
`

	serviceFileTmpl = tmplFileHeader + `

import (
	"github.com/gin-gonic/gin"
)

{{range .Methods}}

{{if .PathVars}}
// {{.Name}}PathVars is the path variables of {{.Name}}.
type {{.Name}}PathVars struct {
{{range .PathVars -}}
	{{.Field}} {{.Type}}
{{end}}
}
{{end}}

{{if .Queries}}
// {{.Name}}Queries is the query parameters of {{.Name}}.
type {{.Name}}Queries struct {
{{range .Queries -}}
	{{.Field}} {{.Kind}} ` + "`form:\"{{.Name}}\"`" + `
{{end}}
}
{{end}}

{{end}}

// {{.Name}} {{.Comment}}
type {{.Name}} interface {
{{range .Methods -}}
	// {{.Name}} {{.Comment}}
	{{.Name}}(
		{{- if .PathVars}}vars {{.Name}}PathVars,{{end -}}
		{{- if .Queries}}q {{.Name}}Queries,{{end -}}
		{{- with .RequestBody}}req {{.}},{{end -}}
	) ({{.Response}}, error)
{{end}}
}

// Set{{.Name}} sets the current service instance.
func Set{{.Name}}(service {{.Name}}, handlers ...gin.HandlerFunc) {
	default{{.Name}} = service
}

type todo{{.Name}} struct{}

{{range .Methods}}
func (todo{{$.Name}}) {{.Name}}(
	{{- if .PathVars}}{{.Name}}PathVars,{{end -}}
	{{- if .Queries}}{{.Name}}Queries,{{end -}}
	{{- with .RequestBody}}{{.}},{{end -}}
) ({{.Response}}, error) {
	panic("not implemented")
}
{{end}}

func new{{.Name}}Routes(r *gin.Engine) *gin.Engine {
	for _, registry := range default{{.Name}}Registry {
		var handlers []gin.HandlerFunc
		for _, h := range registry.PreMain {
			handlers = append(handlers, h)
		}
		handlers = append(handlers, registry.Main)
		for _, h := range registry.PostMain {
			handlers = append(handlers, h)
		}
		r.Handle(registry.HttpMethod, registry.URL, handlers...)
	}
	return r
}

var (
	default{{.Name}} {{.Name}} = todo{{.Name}}{}

	default{{.Name}}Registry = map[string]*ginRegistry{
{{range .Methods -}}
			{{.Name | printf "%q"}}: {
			HttpMethod: {{.HttpMethod | printf "%q"}},
			URL: {{.Path | printf "%q"}},
		},
{{end}}
	}
)
`

	routerFileTmpl = tmplFileHeader + `

import (
	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	r := gin.Default()
	return r
}
`
)

var (
	CodegenModes = map[string]struct{}{
		CodegenModeServer: {},
		CodegenModeClient: {},
	}

	ErrCodegenInpathNotExists = errors.New("input path not exists")
	ErrCodegenInpathNotDir    = errors.New("input path not a directory")
)

type Codegen struct {
	*Parser

	outpath string
	mode    string
}

func NewCodegen() *Codegen {
	return &Codegen{
		Parser: NewParser(),
	}
}

func (c *Codegen) MkOutpath() error {
	inpath := c.inpath

	info, err := os.Stat(inpath)
	if os.IsNotExist(err) {
		return fmt.Errorf("%w: %s", ErrCodegenInpathNotExists, inpath)
	}
	if !info.IsDir() {
		return fmt.Errorf("%w: %s", ErrCodegenInpathNotDir, inpath)
	}

	c.outpath = filepath.Join(inpath, "ginapi")
	_ = os.MkdirAll(c.outpath, os.ModePerm)

	return nil
}

func (c *Codegen) Run() error {
	if err := c.MkOutpath(); err != nil {
		return err
	}
	if err := c.Parser.Parse(); err != nil {
		return err
	}
	if err := c.Generate(); err != nil {
		return err
	}
	return nil
}

func (c *Codegen) Generate() error {
	if err := c.generateServices(); err != nil {
		return err
	}
	if err := c.generateCommon(); err != nil {
		return err
	}
	if err := c.copyModels(); err != nil {
		return err
	}
	return nil
}

func (c *Codegen) generateServices() error {
	for _, service := range c.Services {
		tmpl, err := template.New("ginapi").Parse(serviceFileTmpl)
		if err != nil {
			return err
		}

		buf := bytes.NewBufferString("")
		if err := tmpl.Execute(buf, service); err != nil {
			return fmt.Errorf("text/template: %w", err)
		}

		output, err := format.Source(buf.Bytes())
		if err != nil {
			return fmt.Errorf("gofmt: %w", err)
		}

		outpath := filepath.Join(c.outpath, service.Filepath)
		if err := ioutil.WriteFile(outpath, output, filePerm); err != nil {
			return err
		}
	}
	return nil
}

func (c *Codegen) generateCommon() error {
	output, err := format.Source([]byte(commonFileTmpl))
	if err != nil {
		return fmt.Errorf("gofmt: %w", err)
	}
	outpath := filepath.Join(c.outpath, "common.go")
	if err := ioutil.WriteFile(outpath, output, filePerm); err != nil {
		return err
	}
	return nil
}

func (c *Codegen) copyModels() error {
	for _, path := range c.Parser.modelPaths {
		indata, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		indata = bytes.ReplaceAll(indata, []byte("package openapi"), []byte("package ginapi"))

		outpath := filepath.Join(c.outpath, filepath.Base(path))
		if err := ioutil.WriteFile(outpath, indata, filePerm); err != nil {
			return err
		}
	}
	return nil
}
