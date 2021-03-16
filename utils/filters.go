package ginapiutil

import (
	"context"
	"io/ioutil"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/gin-gonic/gin"
	"github.com/rakyll/statik/fs"
)

var defaultFilter *openapi3filter.Router

func initFilters(filename string) {
	sfs, err := fs.New()
	if err != nil {
		panic(err)
	}

	f, err := sfs.Open(filename)
	if err != nil {
		panic(err)
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}

	swagger, err := openapi3.NewSwaggerLoader().LoadSwaggerFromData(data)
	if err != nil {
		panic(err)
	}

	defaultFilter = openapi3filter.NewRouter().WithSwagger(swagger)
}

// MustValidateRequest validates the upcoming request, panics when it fails,
// user could recover from the panic and check if it's a schema violation error:
// `openapi3filter.RequestError`.
func MustValidateRequest(c *gin.Context) {
	ctx := context.Background()
	req := c.Request

	route, pathParams, err := defaultFilter.FindRoute(req.Method, req.URL)
	if err != nil {
		panic(err)
	}

	input := &openapi3filter.RequestValidationInput{
		Request:     req,
		PathParams:  pathParams,
		QueryParams: req.Form,
		Route:       route,
	}

	if err := openapi3filter.ValidateRequest(ctx, input); err != nil {
		panic(err)
	}
}

// UseValidation initializes a default filter, with a given filename for the
// OpenAPI document in `http.Filesystem`, and do the validation as a middleware.
func UseValidation(filename string) gin.HandlerFunc {
	initFilters(filename)

	return func(c *gin.Context) {
		MustValidateRequest(c)
		c.Next()
	}
}
