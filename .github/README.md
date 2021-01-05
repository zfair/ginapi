# ginapi

Ginapi is a dead simple code generator for generating server/client code from your OpenAPI files, targeting the Gin web
framework.

## Ideas

* Just adapt the results from `go-gin-server` target of `openapi-generator-cli`, so:
    * Parse the generated code with `go/parser`
    * Parse the YAML/JSON specs to complement the `requestBodies`/`responses`
    * Specs are only for complementary purposes, don't rely on it too much
    * Unnecessary to have the generator installed to run this
* Leave the generated code untouched
* Use interfaces and default *not-implemented* methods for further implementation
* Provide better ways to register handlers and routers, in case of middlewares
* Use [hashicorp/go-retryablehttp] for client code

[hashicorp/go-retryablehttp]: https://github.com/hashicorp/go-retryablehttp

## License

MIT
