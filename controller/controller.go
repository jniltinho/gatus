package controller

import (
	"os"
	"time"

	"gatus/v5/api"
	"gatus/v5/config"
	"gatus/v5/liveupdates"
	"github.com/TwiN/logr"
	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

// shutdownTimeout is how long Shutdown waits for the open connections, e.g. an event stream blocked by a slow client
const shutdownTimeout = 10 * time.Second

var (
	app *fiber.App
)

// Handle creates the router and starts the server
func Handle(cfg *config.Config) {
	api := api.New(cfg)
	app = api.Router()
	server := app.Server()
	server.ReadTimeout = 15 * time.Second
	server.WriteTimeout = 15 * time.Second
	server.IdleTimeout = 15 * time.Second
	// Fork: the write timeout applies to the whole response, so the event streams get a longer one, decided before
	// routing from the path of the request
	server.HeaderReceived = eventStreamRequestConfig
	if os.Getenv("ROUTER_TEST") == "true" {
		return
	}
	logr.Info("[controller.Handle] Listening on " + cfg.Web.SocketAddress())
	if cfg.Web.HasTLS() {
		err := app.ListenTLS(cfg.Web.SocketAddress(), cfg.Web.TLS.CertificateFile, cfg.Web.TLS.PrivateKeyFile)
		if err != nil {
			logr.Fatalf("[controller.Handle] %s", err.Error())
		}
	} else {
		err := app.Listen(cfg.Web.SocketAddress())
		if err != nil {
			logr.Fatalf("[controller.Handle] %s", err.Error())
		}
	}
	logr.Info("[controller.Handle] Server has shut down successfully")
}

// Shutdown stops the server
func Shutdown() {
	if app != nil {
		_ = app.ShutdownWithTimeout(shutdownTimeout)
		app = nil
	}
}

// eventStreamRequestConfig gives the requests of the event streams a write timeout longer than their maximum duration.
// The other requests keep the timeouts of the server, which an empty RequestConfig does not change.
func eventStreamRequestConfig(header *fasthttp.RequestHeader) fasthttp.RequestConfig {
	if liveupdates.IsEventsPath(string(header.RequestURI())) {
		return fasthttp.RequestConfig{WriteTimeout: liveupdates.StreamWriteTimeout}
	}
	return fasthttp.RequestConfig{}
}
