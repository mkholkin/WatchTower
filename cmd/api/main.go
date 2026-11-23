package main

import (
	"WatchTower/configs"
	apigen "WatchTower/internal/api/http/v1/gen"
	"WatchTower/internal/api/http/v1/handler"
	"WatchTower/internal/infra/rdap"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"context"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"WatchTower/internal/app"
	"WatchTower/internal/observability"
)

const swaggerHTML = `<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
    <meta charset="UTF-8">
    <title>ServiceDesk Service Swagger</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@3.12.1/swagger-ui.css">


</head>
<body>

<div id="swagger-ui"></div>

<script src="https://unpkg.com/swagger-ui-dist@3.12.1/swagger-ui-standalone-preset.js"></script>
<script src="https://unpkg.com/swagger-ui-dist@3.12.1/swagger-ui-bundle.js"></script>

<script>

    window.onload = function() {
        // Build a system
        const ui = SwaggerUIBundle({
            urls: [{url: "/openapi.yaml", name: "WatchTower API"}, {url: "/openapi-mfa.yaml", name: "MFA authentication"}, {url: "/openapi-rdap.yaml", name: "Domain registration"}],
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
	r.StaticFile("/openapi.yaml", "./api/openapi.yaml")
	r.StaticFile("/openapi-mfa.yaml", "./api/mfa.openapi.yaml")
	r.StaticFile("/openapi-rdap.yaml", "./api/rdap.openapi.yaml")
	r.GET("/swagger", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerHTML))
	})
}

func setupCORS(r *gin.Engine) {
	corsCfg := cors.DefaultConfig()
	corsCfg.AllowAllOrigins = true
	corsCfg.AddAllowHeaders("Authorization")
	corsCfg.AddAllowMethods("GET", "POST", "OPTIONS", "PUT", "DELETE", "PATCH")
	corsMiddleware := cors.New(corsCfg)

	r.Use(corsMiddleware)
}

func SetupRouter(handler *handler.ApiHandler, middleware ...gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	r.ContextWithFallback = true
	r.Use(middleware...)
	r.Use(gin.Recovery())
	setupCORS(r)

	SetupSwaggerRoute(&r.RouterGroup)

	middlewares := []apigen.StrictMiddlewareFunc{
		handler.AuthStrictMiddleware,
	}

	strictHandlerWrapper := apigen.NewStrictHandler(handler, middlewares)
	apigen.RegisterHandlers(r, strictHandlerWrapper)

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

	recorder := configureRecorder()
	defer closeRecorder(recorder)
	logger, closer, err := configs.NewLoggerFromConfig(cfg.Logging)
	if err != nil {
		log.Fatal("cannot initialize request logger")
	}
	if closer != nil {
		defer closer.Close()
	}
	appObj, err := app.InitApp(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to init app: %v", err)
	}
	defer appObj.Shutdown()

	apiHandler := handler.NewApiHandler(
		appObj.AuthSvc,
		appObj.MonitoringSvc,
		appObj.AlertContactsSvc,
		appObj.MaintenanceSvc,
		appObj.MetricsQuerySvc,
	)

	r := SetupRouter(apiHandler, observability.Middleware(recorder, logger, os.Getenv("WT_HTTP_DEBUG") == "true"))
	if appObj.MFA != nil {
		handler.RegisterMFARoutes(r, appObj.MFA)
	}

	registerRDAP(r, cfg.RDAP, appObj)

	go appObj.Start(ctx)

	port := ":8080"
	log.Printf("Starting API server on port %s", port)
	err = r.Run(port)

	if err != nil {
		log.Fatal(err)
	}
}

func configureRecorder() *observability.Recorder {
	path := os.Getenv("WT_TRACE_FILE")
	if path == "" {
		return nil
	}
	recorder, err := observability.New(path)
	if err != nil {
		log.Fatal("cannot open trace output")
	}
	observability.SetCurrent(recorder)
	return recorder
}

func closeRecorder(recorder *observability.Recorder) {
	if recorder == nil {
		return
	}
	if err := recorder.Close(); err != nil {
		log.Print(err)
	}
}

func configureRDAP(cfg configs.RDAPConfig) (handler.DomainLookup, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	timeout := cfg.TimeoutMS
	if timeout == 0 {
		timeout = 5000
	}
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://rdap.verisign.com/com/v1"
	}
	return rdap.New(baseURL, time.Duration(timeout)*time.Millisecond)
}

func registerRDAP(router gin.IRouter, cfg configs.RDAPConfig, application *app.App) {
	client, err := configureRDAP(cfg)
	if err != nil {
		log.Fatal("invalid RDAP configuration")
	}
	handler.RegisterRDAPRoutes(router, application.AuthSvc, application.MonitoringSvc, client)
}
