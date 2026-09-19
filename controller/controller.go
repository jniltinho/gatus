package controller

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"gatus/v5/api"
	"gatus/v5/config"
	"github.com/TwiN/logr"
)

const (
	// shutdownTimeout is how long Shutdown waits for the open connections, e.g. an event stream blocked by a slow client.
	// Past it the connections are closed: http.Server.Shutdown alone never closes an active one.
	shutdownTimeout = 10 * time.Second

	// serverTimeout is the read, write and idle timeout of the ordinary requests. The event streams give themselves a
	// longer write deadline, per request, before their first byte (see api/live_updates.go).
	serverTimeout = 15 * time.Second
)

var (
	mutex  sync.Mutex
	server *http.Server
	// stopped is closed when the server of the current cycle has stopped serving
	stopped chan struct{}
)

// Handle creates the router and starts the server. It blocks until the server stops, and a stop asked through Shutdown
// is not an error: a reload of the configuration stops the server and starts another one.
func Handle(cfg *config.Config) {
	router := api.New(cfg).Router()
	// A new server for every cycle: an http.Server cannot be used again after Shutdown
	current := &http.Server{
		Addr:              cfg.Web.SocketAddress(),
		Handler:           router,
		ReadTimeout:       serverTimeout,
		ReadHeaderTimeout: serverTimeout,
		WriteTimeout:      serverTimeout,
		IdleTimeout:       serverTimeout,
		// web.read-buffer-size used to be the read buffer of fasthttp, which also capped the headers. net/http counts them
		// in another way and has a margin of its own, so this is an approximation of the same limit, not the same number.
		MaxHeaderBytes: cfg.Web.ReadBufferSize,
	}
	done := make(chan struct{})
	mutex.Lock()
	server, stopped = current, done
	mutex.Unlock()
	defer close(done)
	if os.Getenv("ROUTER_TEST") == "true" {
		return
	}
	listener, err := net.Listen("tcp", current.Addr)
	if err != nil {
		logr.Fatalf("[controller.Handle] %s", err.Error())
	}
	logr.Info("[controller.Handle] Listening on " + cfg.Web.SocketAddress())
	if cfg.Web.HasTLS() {
		err = current.ServeTLS(listener, cfg.Web.TLS.CertificateFile, cfg.Web.TLS.PrivateKeyFile)
	} else {
		err = current.Serve(listener)
	}
	// Shutdown and Close make Serve return http.ErrServerClosed, which is how a server is meant to stop
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logr.Fatalf("[controller.Handle] %s", err.Error())
	}
	logr.Info("[controller.Handle] Server has shut down successfully")
}

// Shutdown stops the server and returns once it has stopped serving, so that the next cycle can listen on the same
// address. The event streams must be closed before it (liveupdates.Close), otherwise it waits for them.
func Shutdown() {
	mutex.Lock()
	current, done := server, stopped
	server, stopped = nil, nil
	mutex.Unlock()
	if current == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := current.Shutdown(ctx); err != nil {
		// The deadline passed with connections still active: they are closed, as ShutdownWithTimeout of Fiber did
		_ = current.Close()
	}
	if done != nil {
		select {
		case <-done:
		case <-time.After(shutdownTimeout):
		}
	}
}
