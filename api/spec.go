package api

import _ "embed"

// OpenAPISpec is the canonical API contract served by Swagger UI.
//
//go:embed openapi.yaml
var OpenAPISpec []byte
