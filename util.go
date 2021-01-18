package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

var (
	ErrUtilUseRef            = errors.New("inline objects not recommended, use $ref instead")
	ErrUtilBadOapiSchemaType = errors.New("bad OAPI schema type")
	ErrUtilBadOapiRef        = errors.New("bad OAPI ref")
)

func OapiRefToGoType(ref string) (string, error) {
	parts := strings.Split(ref, "/")
	l := len(parts)
	if l == 0 {
		return "", fmt.Errorf("%w: %s", ErrUtilBadOapiRef, ref)
	}
	return strings.Title(parts[l-1]), nil
}

func OapiToGoType(ref *openapi3.SchemaRef) (string, error) {
	if ref.Ref != "" {
		t, err := OapiRefToGoType(ref.Ref)
		if err != nil {
			return "", err
		}
		return t, nil
	}

	schema := ref.Value
	t := schema.Type

	switch t {
	case "number":
		switch schema.Format {
		case "float":
			return "float32", nil
		case "double":
			return "float64", nil
		}
	case "integer":
		switch f := schema.Format; f {
		case "int32":
			fallthrough
		case "int64":
			return f, nil
		}
	case "string":
		return t, nil
	case "boolean":
		return "bool", nil
	case "array":
		tt, err := OapiToGoType(schema.Items)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("[]%s", tt), nil
	case "object":
		if schema.Properties != nil {
			return "", ErrUtilUseRef
		}
	}

	return "", fmt.Errorf("%w: %s", ErrUtilBadOapiSchemaType, schema.Type)
}
