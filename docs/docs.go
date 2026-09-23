package docs

import _ "embed"

// OpenAPISpec contains the raw OpenAPI 3.0 specification in YAML format.
//
//go:embed openapi.yaml
var OpenAPISpec []byte

// DocsHTML contains the Scalar API Reference interactive documentation HTML template.
//
//go:embed docs.html
var DocsHTML []byte
