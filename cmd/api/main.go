package main

import (
	apispec "WatchTower/api"
	"WatchTower/configs"
	v1gen "WatchTower/internal/api/http/v1/gen"
	v1handler "WatchTower/internal/api/http/v1/handler"
	v2gen "WatchTower/internal/api/http/v2/gen"
	v2handler "WatchTower/internal/api/http/v2/handler"
	"fmt"
	"log"
	"net/http"
	"os"

	"context"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"WatchTower/internal/app"
)

const swaggerHTML = `<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
    <meta charset="UTF-8">
	<title>WatchTower API</title>
	<link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">


</head>
<body>

<div id="swagger-ui"></div>

<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>

<script>

    window.onload = function() {
        // Build a system
        const ui = SwaggerUIBundle({
            url: "/openapi.yaml",
            dom_id: '#swagger-ui',
            deepLinking: true,
            presets: [
                SwaggerUIBundle.presets.apis,
                SwaggerUIStandalonePreset
            ],
            plugins: [
                SwaggerUIBundle.plugins.DownloadUrl
            ],
            layout: "StandaloneLayout",
        })
        window.ui = ui
    }
</script>
</body>
</html>`

func SetupSwaggerRoute(r *gin.RouterGroup) {
	r.GET("/openapi.yaml", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/yaml; charset=utf-8", apispec.OpenAPISpec)
	})
	r.GET("/swagger", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerHTML))
	})
}

func setupCORS(r *gin.Engine) {
	corsCfg := cors.DefaultConfig()
	corsCfg.AllowAllOrigins = true
	corsCfg.AddAllowHeaders("Authorization")
	corsCfg.AddExposeHeaders("Location", "WWW-Authenticate")
	corsCfg.AddAllowMethods("GET", "POST", "OPTIONS", "PUT", "DELETE", "PATCH")
	corsMiddleware := cors.New(corsCfg)

	r.Use(corsMiddleware)
}

func SetupRouter(v1Handler *v1handler.ApiHandler, v2Handler *v2handler.APIHandler) *gin.Engine {
	gin.EnableJsonDecoderDisallowUnknownFields()
	r := gin.Default()
	setupCORS(r)

	SetupSwaggerRoute(&r.RouterGroup)

	v1Middlewares := []v1gen.StrictMiddlewareFunc{
		v1Handler.AuthStrictMiddleware,
	}
	v1StrictHandler := v1gen.NewStrictHandler(v1Handler, v1Middlewares)
	// Keep unversioned routes for the existing GUI and expose the same legacy
	// contract explicitly as v1 for external clients.
	v1gen.RegisterHandlers(r, v1StrictHandler)
	v1gen.RegisterHandlersWithOptions(r, v1StrictHandler, v1gen.GinServerOptions{BaseURL: "/api/v1"})

	v2Middlewares := []v2gen.StrictMiddlewareFunc{
		v2Handler.AuthStrictMiddleware,
	}
	v2StrictHandler := v2gen.NewStrictHandlerWithOptions(v2Handler, v2Middlewares, v2gen.StrictGinServerOptions{
		RequestErrorHandlerFunc: func(c *gin.Context, err error) {
			c.JSON(http.StatusBadRequest, v2gen.Error{Code: "invalid_request", Message: err.Error()})
		},
		HandlerErrorFunc: func(c *gin.Context, err error) {
			c.JSON(http.StatusInternalServerError, v2gen.Error{Code: "internal_error", Message: "an unexpected error occurred"})
		},
		ResponseErrorHandlerFunc: func(c *gin.Context, err error) {
			c.JSON(http.StatusInternalServerError, v2gen.Error{Code: "internal_error", Message: "an unexpected error occurred"})
		},
	})
	v2gen.RegisterHandlers(r, v2StrictHandler)

	return r
}

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <path-to-config.yaml>\n", os.Args[0])
		return
	}

	configPath := os.Args[1]
	cfg, err := configs.Load(configPath)
	if err != nil {
		log.Fatalf("failed to load config %q: %v", configPath, err)
	}

	appObj, err := app.InitApp(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to init app: %v", err)
	}
	defer appObj.Shutdown()

	v1APIHandler := v1handler.NewApiHandler(
		appObj.AuthSvc,
		appObj.MonitoringSvc,
		appObj.AlertContactsSvc,
		appObj.MaintenanceSvc,
		appObj.MetricsQuerySvc,
	)
	v2APIHandler := v2handler.NewApiHandler(
		appObj.AuthSvc,
		appObj.MonitoringSvc,
		appObj.AlertContactsSvc,
		appObj.MaintenanceSvc,
		appObj.MetricsQuerySvc,
	)

	r := SetupRouter(v1APIHandler, v2APIHandler)

	go appObj.Start(ctx)

	port := ":8080"
	log.Printf("Starting API server on port %s", port)
	err = r.Run(port)

	if err != nil {
		log.Fatal(err)
	}
}
