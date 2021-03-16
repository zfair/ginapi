# Ginapi Examples

Run the following commands to perform code generation.

```bash
$ docker run --rm -v $(PWD):/local \
  openapitools/openapi-generator-cli generate \
    -i /local/petstore.yaml
    -g go-gin-server \
    -o /local/generated
$ ginapi -i generated
$ go generate ./...
```
