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

func OapiTagToServiceName(tag string) string {
	parts := strings.Split(tag, " ")
	for i := 0; i < len(parts); i++ {
		parts[i] = strings.Title(parts[i])
	}
	parts = append(parts, "Service")
	return strings.Join(parts, "")
}

func OapiRefToGoStruct(ref string) (string, error) {
	parts := strings.Split(ref, "/")
	if l := len(parts); l > 0 {
		return strings.Title(parts[l-1]), nil
	}
	return "", fmt.Errorf("%w: %s", ErrUtilBadOapiRef, ref)
}

func OapiRefToGoPtr(ref string) (string, error) {
	ty, err := OapiRefToGoStruct(ref)
	if err != nil {
		return "", err
	}
	return "*" + ty, nil
}

func OapiToGoType(ref *openapi3.SchemaRef) (string, error) {
	if ref.Ref != "" {
		t, err := OapiRefToGoPtr(ref.Ref)
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

func OapiToGinPathParam(param string) (ret string) {
	ret = param
	ret = strings.ReplaceAll(ret, "{", ":")
	ret = strings.ReplaceAll(ret, "}", "")
	return
}
