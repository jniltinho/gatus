package controller

import (
	"bufio"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"gatus/v5/config"
	"gatus/v5/config/endpoint"
	"gatus/v5/config/web"
	"gatus/v5/liveupdates"
	"github.com/gofiber/fiber/v2"
)

func TestHandle(t *testing.T) {
	cfg := &config.Config{
		Web: &web.Config{
			Address: "0.0.0.0",
			Port:    rand.Intn(65534),
		},
		Endpoints: []*endpoint.Endpoint{
			{
				Name:  "frontend",
				Group: "core",
			},
			{
				Name:  "backend",
				Group: "core",
			},
		},
	}
	_ = os.Setenv("ROUTER_TEST", "true")
	_ = os.Setenv("ENVIRONMENT", "dev")
	defer os.Clearenv()
	Handle(cfg)
	defer Shutdown()
	request := httptest.NewRequest("GET", "/health", http.NoBody)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 {
		t.Error("expected GET /health to return status code 200")
	}
	if app == nil {
		t.Fatal("server should've been set (but because we set ROUTER_TEST, it shouldn't have been started)")
	}
}

func TestHandleTLS(t *testing.T) {
	scenarios := []struct {
		name               string
		tls                *web.TLSConfig
		expectedStatusCode int
	}{
		{
			name:               "good-tls-config",
			tls:                &web.TLSConfig{CertificateFile: "../testdata/cert.pem", PrivateKeyFile: "../testdata/cert.key"},
			expectedStatusCode: 200,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			cfg := &config.Config{
				Web: &web.Config{Address: "0.0.0.0", Port: rand.Intn(65534), TLS: scenario.tls},
				Endpoints: []*endpoint.Endpoint{
					{Name: "frontend", Group: "core"},
					{Name: "backend", Group: "core"},
				},
			}
			if err := cfg.Web.ValidateAndSetDefaults(); err != nil {
				t.Error("expected no error from web (TLS) validation, got", err)
			}
			_ = os.Setenv("ROUTER_TEST", "true")
			_ = os.Setenv("ENVIRONMENT", "dev")
			defer os.Clearenv()
			Handle(cfg)
			defer Shutdown()
			request := httptest.NewRequest("GET", "/health", http.NoBody)
			response, err := app.Test(request)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != scenario.expectedStatusCode {
				t.Errorf("%s %s should have returned %d, but returned %d instead", request.Method, request.URL, scenario.expectedStatusCode, response.StatusCode)
			}
			if app == nil {
				t.Fatal("server should've been set (but because we set ROUTER_TEST, it shouldn't have been started)")
			}
		})
	}
}

func TestShutdown(t *testing.T) {
	// Pretend that we called controller.Handle(), which initializes the server variable
	app = fiber.New()
	Shutdown()
	if app != nil {
		t.Error("server should've been shut down")
	}
}

func TestEventStreamRequestConfig(t *testing.T) {
	previousWriteTimeout := liveupdates.StreamWriteTimeout
	liveupdates.StreamWriteTimeout = 5 * time.Second
	defer func() { liveupdates.StreamWriteTimeout = previousWriteTimeout }()
	app := fiber.New()
	slowStream := func(c *fiber.Ctx) error {
		c.Context().SetBodyStreamWriter(func(writer *bufio.Writer) {
			for i := 0; i < 6; i++ {
				_, _ = writer.WriteString("data: {}\n\n")
				_ = writer.Flush()
				time.Sleep(100 * time.Millisecond)
			}
		})
		return nil
	}
	app.Get("/api/v1/endpoints/:key/events", slowStream)
	app.Get("/api/v1/endpoints/statuses/events-export", slowStream)
	server := app.Server()
	server.WriteTimeout = 250 * time.Millisecond
	server.HeaderReceived = eventStreamRequestConfig
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = app.Listener(listener) }()
	defer func() { _ = app.ShutdownWithTimeout(time.Second) }()
	base := "http://" + listener.Addr().String()
	read := func(path string) (string, error) {
		response, err := http.Get(base + path)
		if err != nil {
			return "", err
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		return string(body), err
	}
	for _, path := range []string{"/api/v1/endpoints/jobs_backup/events", "/api/v1/endpoints/jobs_backup/events?lastEventId=3"} {
		if body, err := read(path); err != nil || strings.Count(body, "data: {}") != 6 {
			t.Errorf("%s: expected the whole stream with the longer write timeout, got %q %v", path, body, err)
		}
	}
	if body, err := read("/api/v1/endpoints/statuses/events-export"); err == nil && strings.Count(body, "data: {}") == 6 {
		t.Errorf("expected another route to keep the default write timeout, got the whole body %q", body)
	}
}
