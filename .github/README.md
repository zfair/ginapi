# ginapi

Ginapi is an opinionated code generator for **server code only** from your OpenAPI files, targeting the [Gin]
web framework.

[Gin]: https://github.com/gin-gonic/gin

## Motivating example

For the classical [Pet Store] example, use the Ginapi generated code like this:

[Pet Store]: ../examples/petstore.yaml

```go
package main

import (
	ginapiutil "github.com/anqur/ginapi/utils"
	"github.com/gin-gonic/gin"

	"github.com/anqur/ginapi/examples/generated/ginapi"
	_ "github.com/anqur/ginapi/examples/generated/statik"
)

// Run `go generate ./...` to:
// * Generate basic code by openapi-generator-cli
// * Do the Ginapi code generation
// * Use statik to serve the OpenAPI file

//go:generate docker run --rm -v $PWD:/local openapitools/openapi-generator-cli generate -i /local/petstore.yaml -g go-gin-server -o /local/generated
//go:generate ginapi -i generated -vars {"server":"http://localhost:8088"}
//go:generate statik -src=. -dest=./generated -include=petstore.yaml
func main() {
	// Register our implementations and some middlewares.
	ginapi.RegisterPetsService(
		&DefaultPetsService{},

		// Optionally, we can validate requests by the OpenAPI file.
		ginapiutil.UseValidation("/petstore.yaml"),
	)

	// Let's serve it.
	r := ginapi.Initialize(gin.Default())
	if err := r.Run("localhost:8088"); err != nil {
		panic(err)
	}
}

type DefaultPetsService struct{}

// Methods with queries/path variables/request bodies as arguments, responses as
// return value. Not just empty handler functions :(.
func (p *DefaultPetsService) CreatePets(h ginapi.CreatePetsHeaders) (*ginapi.Result, error) {
	panic("TODO")
}

func (p *DefaultPetsService) ListPets(q ginapi.ListPetsQueries) (*ginapi.Pets, error) {
	panic("TODO")
}

func (p *DefaultPetsService) ShowPetById(vars ginapi.ShowPetByIdPathVars) (*ginapi.Pet, error) {
	panic("TODO")
}

func (p *DefaultPetsService) DeletePet(vars ginapi.DeletePetPathVars) error {
	panic("TODO")
}
```

## How is it opinionated?

* Reuse the `go-gin-server` target of [openapi-generator-cli] for generated models and canonicalized OpenAPI files
* We hate empty handler functions ❌, we need interfaces and type safety! ✅
* Provide better ways to register handlers and routers, in case of middlewares

[openapi-generator-cli]: https://github.com/OpenAPITools/openapi-generator-cli

## More examples

See [examples](../examples).

## License

MIT
