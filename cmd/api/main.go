package main

import (
	"WatchTower/configs"
	apigen "WatchTower/internal/api/http/v1/gen"
	"WatchTower/internal/api/http/v1/handler"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
            urls: [{url: "/openapi.yaml", name: "WatchTower API"}, {url: "/openapi-mfa.yaml", name: "MFA authentication"}],
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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) (result error) {
	if len(args) < 2 {
		return fmt.Errorf("usage: %s <path-to-config.yaml>", args[0])
	}
	cfg, err := configs.Load(args[1])
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	recorder, err := setupTracing(ctx)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, shutdownTracing(recorder)) }()
	logger, closer, err := configs.NewLoggerFromConfig(cfg.Logging)
	if err != nil {
		return errors.New("cannot initialize request logger")
	}
	if closer != nil {
		defer closer.Close()
	}
	appObj, err := app.InitApp(ctx, cfg)
	if err != nil {
		return fmt.Errorf("initialize application: %w", err)
	}
	defer appObj.Shutdown()
	apiHandler := handler.NewApiHandler(appObj.AuthSvc, appObj.MonitoringSvc, appObj.AlertContactsSvc, appObj.MaintenanceSvc, appObj.MetricsQuerySvc)
	router := SetupRouter(apiHandler, observability.Middleware(recorder, logger, os.Getenv("WT_HTTP_DEBUG") == "true"))
	if appObj.MFA != nil {
		handler.RegisterMFARoutes(router, appObj.MFA)
	}
	appObj.Start(ctx)
	server := &http.Server{Addr: ":8080", Handler: router, ReadHeaderTimeout: 10 * time.Second}
	log.Printf("Starting API server on port %s", server.Addr)
	return serve(ctx, server)
}

func setupTracing(ctx context.Context) (*observability.Recorder, error) {
	if os.Getenv("WT_TRACING_ENABLED") != "true" {
		return nil, nil
	}
	recorder, err := observability.New(ctx)
	if err != nil {
		return nil, errors.New("cannot initialize tracing")
	}
	observability.SetCurrent(recorder)
	return recorder, nil
}

func shutdownTracing(recorder *observability.Recorder) error {
	observability.SetCurrent(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := recorder.Shutdown(ctx)
	summary, _ := json.Marshal(recorder.Summary())
	fmt.Fprintf(os.Stderr, "OTEL_EXPORT_SUMMARY %s\n", summary)
	return err
}

func serve(ctx context.Context, server *http.Server) error {
	stopped := make(chan error, 1)
	go func() { stopped <- server.ListenAndServe() }()
	select {
	case err := <-stopped:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return errors.Join(err, server.Close())
		}
		err := <-stopped
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
