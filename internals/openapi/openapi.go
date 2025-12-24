package openapi

import (
	"encoding/json"
	"fmt"
	"log"
	"myapp/internals/router"
	"reflect"
	"strings"
)

func goTypeToOpenAPI(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "integer"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Struct:
		return "object"
	default:
		log.Printf("[WARNING]: failed to convert type %s to OpenAPI variant.", t.Kind())
		return "string"
	}
}

func schemaFromStruct(t reflect.Type) map[string]interface{} {
	schema := map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
	if t.Kind() == reflect.Struct {
		props := map[string]interface{}{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			jsonTag := f.Tag.Get("json")
			if jsonTag == "" {
				jsonTag = f.Name
			}
			props[jsonTag] = map[string]string{"type": goTypeToOpenAPI(f.Type)}
		}
		schema["properties"] = props
	}
	return schema
}

func generateSwaggerDocMap(routes []router.Route) map[string]interface{} {
	openapi := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]string{
			"title":   "Auto API",
			"version": "1.0.0",
		},
		"paths": map[string]interface{}{},
	}

	for _, route := range routes {
		op := map[string]interface{}{
			"summary":   route.Path,
			"responses": map[string]interface{}{"200": map[string]interface{}{"description": "OK"}},
		}

		handlerType := reflect.TypeOf(route.Handler)
		if handlerType.Kind() != reflect.Func || handlerType.NumIn() != 1 || handlerType.NumOut() < 1 {
			continue
		}

		reqType := handlerType.In(0)  // Request[Path, Query, Body]
		resType := handlerType.Out(0) // Response

		var pathSchema, querySchema, bodySchema map[string]interface{}

		if reqType.Kind() == reflect.Struct {
			// Path
			if f, ok := reqType.FieldByName("Path"); ok {
				pathSchema = schemaFromStruct(f.Type)
			}
			// Query
			if f, ok := reqType.FieldByName("Query"); ok {
				querySchema = schemaFromStruct(f.Type)
			}
			// Body
			if f, ok := reqType.FieldByName("Body"); ok {
				bodySchema = schemaFromStruct(f.Type)
			}
		}

		// --- Path parameters ---
		parameters := []map[string]interface{}{}
		if pathSchema != nil {
			for name, prop := range pathSchema["properties"].(map[string]interface{}) {
				parameters = append(parameters, map[string]interface{}{
					"name":     name,
					"in":       "path",
					"required": true,
					"schema":   prop,
				})
			}
		}

		// --- Query parameters ---
		if querySchema != nil {
			for name, prop := range querySchema["properties"].(map[string]interface{}) {
				parameters = append(parameters, map[string]interface{}{
					"name":     name,
					"in":       "query",
					"required": false,
					"schema":   prop,
				})
			}
		}
		if len(parameters) > 0 {
			op["parameters"] = parameters
		}

		// --- Request body ---
		if bodySchema != nil {
			op["requestBody"] = map[string]interface{}{
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": bodySchema,
					},
				},
			}
		}

		// --- Response ---
		if resType != nil {
			op["responses"].(map[string]interface{})["200"] = map[string]interface{}{
				"description": "OK",
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": schemaFromStruct(resType),
					},
				},
			}
		}

		openapi["paths"].(map[string]interface{})[route.Path] = map[string]interface{}{
			strings.ToLower(route.Method): op,
		}
	}

	return openapi
}

func GenerateJsonString(routes []router.Route) string {
	openapiMap := generateSwaggerDocMap(routes)

	openapiJSON, err := json.Marshal(openapiMap)
	if err != nil {
		openapiJSON = []byte(`{}`)
	}
	return string(openapiJSON)
}

func GenerateSwaggerDocHtml(routes []router.Route) string {
	openapiJson := GenerateJsonString(routes)
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
  <head>
    <meta charset="UTF-8">
    <title>Swagger UI</title>
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.29.0/swagger-ui.min.css">
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.29.0/swagger-ui-bundle.min.js"></script>
    <script defer>
      const spec = %s;
      const ui = SwaggerUIBundle({
        spec: spec,
        dom_id: '#swagger-ui',
        presets: [
          SwaggerUIBundle.presets.apis,
          SwaggerUIBundle.SwaggerUIStandalonePreset
        ]
      });
    </script>
  </body>
</html>`, string(openapiJson))
}
