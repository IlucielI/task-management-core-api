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
)

//go:embed routes.yaml
var routesYAML []byte

type routeItem struct {
	Method  string `yaml:"method"`
	Path    string `yaml:"path"`
	Handler string `yaml:"handler"`
}

type routeConfig struct {
	Routes []routeItem `yaml:"routes"`
}

func NewRouter(cfg config.Config) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery(), gin.Logger())

	// Central controllers container
	ctrls := controllers.New(cfg)
	ctrlsVal := reflect.ValueOf(ctrls)

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

		httpMethod := strings.ToUpper(strings.TrimSpace(r.Method))
		router.Handle(httpMethod, r.Path, fn)
	}

	return router
}
