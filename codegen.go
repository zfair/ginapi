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
	filePerm = 0644

	tmplFileHeader = `// Generated by ginapi. DO NOT EDIT.
package ginapi
`

	modelFileTmpl = tmplFileHeader + `
{{range .Typedefs}}
type {{.Target}} {{.Source}}
{{end}}
`

	serviceFileTmpl = tmplFileHeader + `

import (
	"net/http"

	"github.com/anqur/ginapi/utils/detail"

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
	{{.Field}} {{.Type}} ` + "`form:\"{{.Name}}\"`" + `
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
	) {{if .Response}} ({{.Response}}, error) {{else}} error {{end}}
{{end}}
}

// Register{{.Name}} registers the current service instance with middlewares.
func Register{{.Name}}(service {{.Name}}, handlers ...gin.HandlerFunc) {
	default{{.Name}} = service
	default{{.Name}}Handlers = handlers
}

{{range .Methods}}
// With{{.Name}} registers middlewares specifically for {{.Name}}.
func With{{.Name}}(handlers ...gin.HandlerFunc) {
	default{{$.Name}}Registry[{{.Name | printf "%q"}}].Middlewares = handlers
}
{{end}}

type todo{{.Name}} struct{}

{{range .Methods}}
func (todo{{$.Name}}) {{.Name}}(
	{{- if .PathVars}}{{.Name}}PathVars,{{end -}}
	{{- if .Queries}}{{.Name}}Queries,{{end -}}
	{{- with .RequestBody}}{{.}},{{end -}}
) {{if .Response}} ({{.Response}}, error) {{else}} error {{end}} {
	panic("not implemented")
}
{{end}}

{{range .Methods}}
func defaultHandle{{.Name}}(c *gin.Context) {
	var err error

{{if .PathVars -}}
	vars := {{.Name}}PathVars{}
{{range .PathVars -}}
	v{{.Field}}, err := detail.{{.Binder}}(c, {{.Name | printf "%q"}})
	if err != nil {
		panic(err)
	}
	vars.{{.Field}} = v{{.Field}}
{{end}}
{{end}}

{{if .Queries}}
	q := {{.Name}}Queries{}
	if err := c.ShouldBind(&q); err != nil {
		panic(err)
	}
{{end}}

{{with .RequestBody}}
	req := {{.}}{}
	if err := c.ShouldBind(&req); err != nil {
		panic(err)
	}
{{end}}

	{{if .Response}}resp, err := {{else}} err = {{end}} default{{$.Name}}.{{.Name}}(
{{if .PathVars -}}
		vars,
{{end -}}
{{if .Queries -}}
		q,
{{end -}}
{{with .RequestBody -}}
		req,
{{end -}}
	)

	if err != nil {
		panic(err)
	}

{{if .Response}}
	c.JSON(http.StatusOK, resp)
{{else}}
	c.Status(http.StatusOK)
{{end}}
}
{{end}}

func new{{.Name}}Routers(r *gin.Engine) *gin.Engine {
	for _, registry := range default{{.Name}}Registry {
		var handlers []gin.HandlerFunc

		for _, h := range default{{.Name}}Handlers {
			handlers = append(handlers, h)
		}

		for _, h := range registry.Middlewares {
			handlers = append(handlers, h)
		}

		handlers = append(handlers, registry.Main)

		r.Handle(registry.HttpMethod, registry.URL, handlers...)
	}
	return r
}

var (
	default{{.Name}} {{.Name}} = todo{{.Name}}{}

	default{{.Name}}Handlers []gin.HandlerFunc

	default{{.Name}}Registry = map[string]*detail.GinRegistry{
{{range .Methods -}}
			{{.Name | printf "%q"}}: {
			HttpMethod: {{.HttpMethod | printf "%q"}},
			URL: {{.Path | printf "%q"}},
			Main: defaultHandle{{.Name}},
		},
{{end}}
	}
)
`

	routerFileTmpl = tmplFileHeader + `

import (
	"github.com/gin-gonic/gin"
)

// Initialize initializes the main router for all services.
func Initialize(r *gin.Engine) *gin.Engine {
{{range .Services}}
	r = new{{.Name}}Routers(r)
{{end}}
	return r
}
`
)

var (
	ErrCodegenInpathNotExists = errors.New("input path not exists")
	ErrCodegenInpathNotDir    = errors.New("input path not a directory")
)

type Codegen struct {
	*Parser
	outpath string
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
	if err := c.copyModels(); err != nil {
		return err
	}
	if err := c.generateTypedefs(); err != nil {
		return err
	}
	if err := c.generateRouters(); err != nil {
		return err
	}
	return nil
}

func (c *Codegen) generateServices() error {
	for _, service := range c.Services {
		outpath := filepath.Join(c.outpath, service.Filepath)
		if err := formattedRender("ginapi-services", serviceFileTmpl, outpath, service); err != nil {
			return err
		}
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

func (c *Codegen) generateTypedefs() error {
	outpath := filepath.Join(c.outpath, "models.go")
	return formattedRender("ginapi-models", modelFileTmpl, outpath, c.Parser)
}

func (c *Codegen) generateRouters() error {
	outpath := filepath.Join(c.outpath, "routers.go")
	return formattedRender("ginapi-routers", routerFileTmpl, outpath, c.Parser)
}

func formattedRender(name, text, outpath string, data interface{}) error {
	tmpl, err := template.New(name).Parse(text)
	if err != nil {
		return err
	}

	buf := bytes.NewBufferString("")
	if err := tmpl.Execute(buf, data); err != nil {
		return fmt.Errorf("text/template: %s: %w", outpath, err)
	}

	output, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("gofmt: %s: %w", outpath, err)
	}

	if err := ioutil.WriteFile(outpath, output, filePerm); err != nil {
		return err
	}

	return nil
}
