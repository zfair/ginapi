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

func OapiToGoType(ref *openapi3.SchemaRef, required bool) (ret string, err error) {
	if ref.Ref != "" {
		var t string
		t, err = OapiRefToGoStruct(ref.Ref)
		if err != nil {
			return
		}
		ret = t
	} else {
		schema := ref.Value
		t := schema.Type

		switch t {
		case "number":
			switch schema.Format {
			case "float":
				ret = "float32"
			case "double":
				ret = "float64"
			}
		case "integer":
			switch f := schema.Format; f {
			case "int32":
				fallthrough
			case "int64":
				ret = f
			}
		case "string":
			ret = t
		case "boolean":
			ret = "bool"
		case "array":
			var tt string
			tt, err = OapiToGoType(schema.Items, required)
			if err != nil {
				return
			}
			ret = fmt.Sprintf("[]%s", tt)
		case "object":
			if schema.Properties != nil {
				err = ErrUtilUseRef
				return
			}
		default:
			err = fmt.Errorf("%w: %s", ErrUtilBadOapiSchemaType, schema.Type)
			return
		}
	}

	if !required {
		ret = "*" + ret
	}

	return
}

func OapiToGinPathParam(param string) (ret string) {
	ret = param
	ret = strings.ReplaceAll(ret, "{", ":")
	ret = strings.ReplaceAll(ret, "}", "")
	return
}
