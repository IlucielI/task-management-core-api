package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"task-management/docs"
)

// OpenAPISpec serves the raw OpenAPI 3.0 specification in YAML format.
func (c *Controllers) OpenAPISpec(ctx *gin.Context) {
	ctx.Data(http.StatusOK, "application/x-yaml; charset=utf-8", docs.OpenAPISpec)
}

// APIDocs serves the modern Scalar API Reference interactive documentation page.
func (c *Controllers) APIDocs(ctx *gin.Context) {
	ctx.Data(http.StatusOK, "text/html; charset=utf-8", docs.DocsHTML)
}
