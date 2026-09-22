package routes

import (
	_ "embed"
	"log"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"

	"task-management/internal/config"
	"task-management/internal/controllers"
	"task-management/internal/middlewares"
	"task-management/internal/services"
)

//go:embed routes.yaml
var routesYAML []byte

type routeItem struct {
	Method  string `yaml:"method"`
	Path    string `yaml:"path"`
	Handler string `yaml:"handler"`
	Auth    bool   `yaml:"auth,omitempty"`
}

type routeConfig struct {
	Routes []routeItem `yaml:"routes"`
}

// NewRouter constructs the central Gin engine and wires controllers and auth middleware.
func NewRouter(cfg config.Config, ctrls *controllers.Controllers, authValidator ...services.AuthValidator) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery(), gin.Logger(), middlewares.ClientMeta())

	// Central controllers container
	if ctrls == nil {
		ctrls = controllers.New(cfg, nil)
	}
	ctrlsVal := reflect.ValueOf(ctrls)

	// Auth validator for protected routes
	var validator services.AuthValidator
	if len(authValidator) > 0 && authValidator[0] != nil {
		validator = authValidator[0]
	}

	// Load and parse embedded YAML routes
	var rCfg routeConfig
	if err := yaml.Unmarshal(routesYAML, &rCfg); err != nil {
		log.Fatalf("failed to parse routes.yaml: %v", err)
	}

	// Auto-dispatch routes dynamically via reflection (zero manual mapping)
	for _, r := range rCfg.Routes {
		methodName := strings.TrimSpace(r.Handler)
		method := ctrlsVal.MethodByName(methodName)

		if !method.IsValid() {
			log.Fatalf("route configuration error: method '%s' not found on Controllers for route %s %s", methodName, r.Method, r.Path)
		}

		fn, ok := method.Interface().(func(*gin.Context))
		if !ok {
			log.Fatalf("route configuration error: method '%s' must have signature func(*gin.Context)", methodName)
		}

		var handlers []gin.HandlerFunc
		if r.Auth {
			handlers = append(handlers, middlewares.Auth(validator))
		}
		handlers = append(handlers, fn)

		httpMethod := strings.ToUpper(strings.TrimSpace(r.Method))
		router.Handle(httpMethod, r.Path, handlers...)
	}

	return router
}
