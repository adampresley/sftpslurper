package main

import (
	"context"
	"embed"
	"log/slog"
	"net/http"

	"github.com/adampresley/adamgokit/httphelpers"
	"github.com/adampresley/adamgokit/mux"
	"github.com/adampresley/adamgokit/rendering"
	"github.com/adampresley/sftpslurper/cmd/sftpslurper/internal/configuration"
	"github.com/adampresley/sftpslurper/cmd/sftpslurper/internal/home"
	"github.com/adampresley/sftpslurper/cmd/sftpslurper/internal/sftp"
)

var (
	Version string = "development"
	appName string = "sftpslurper"

	//go:embed app
	appFS embed.FS

	/* Services */
	renderer rendering.TemplateRenderer

	/* Controllers */
	homeController home.HomeHandlers
)

func main() {
	config := configuration.LoadConfig(Version)
	setupLogger(&config, Version)

	slog.Info("configuration loaded",
		slog.String("app", appName),
		slog.String("version", Version),
		slog.String("loglevel", config.LogLevel),
		slog.String("host", config.Host),
	)

	slog.Debug("setting up...")

	/*
	 * Setup services
	 */
	renderer = rendering.NewGoTemplateRenderer(rendering.GoTemplateRendererConfig{
		TemplateDir:       "app",
		TemplateExtension: ".html",
		TemplateFS:        appFS,
		LayoutsDir:        "layouts",
		ComponentsDir:     "components",
	})

	/*
	 * Setup controllers
	 */
	homeController = home.NewHomeController(home.HomeControllerConfig{
		Config:   &config,
		Renderer: renderer,
	})

	/*
	 * Setup router and http server
	 */
	slog.Debug("setting up routes...")

	routes := []mux.Route{
		{Path: "GET /heartbeat", HandlerFunc: heartbeat},
		{Path: "GET /", HandlerFunc: homeController.HomePage},
		{Path: "GET /about", HandlerFunc: homeController.AboutPage},
		{Path: "GET /uploads", HandlerFunc: homeController.ServeFile},
		{Path: "GET /preview", HandlerFunc: homeController.PreviewContent},
	}

	routerConfig := mux.RouterConfig{
		Address:              config.Host,
		Debug:                Version == "development",
		ServeStaticContent:   true,
		StaticContentRootDir: "app",
		StaticContentPrefix:  "/static/",
		StaticFS:             appFS,
	}

	m := mux.SetupRouter(routerConfig, routes)
	httpServer, quit := mux.SetupServer(routerConfig, m)

	/*
	 * Start up the SFTP server
	 */
	sftpShutdownCtx, sftpCancel := context.WithCancel(context.Background())
	sftp.StartServer(&config, sftpShutdownCtx)

	/*
	 * Wait for graceful shutdown
	 */
	slog.Info("server started")

	<-quit
	sftpCancel()
	mux.Shutdown(httpServer)
	slog.Info("server stopped")
}

func heartbeat(w http.ResponseWriter, r *http.Request) {
	httphelpers.TextOK(w, "OK")
}
