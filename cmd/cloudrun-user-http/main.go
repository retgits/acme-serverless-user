package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/fasthttp/router"
	"github.com/getsentry/sentry-go"
	sentryfasthttp "github.com/getsentry/sentry-go/fasthttp"
	"github.com/retgits/acme-serverless-user/internal/datastore"
	"github.com/retgits/acme-serverless-user/internal/datastore/mongodb"
	gcrwavefront "github.com/retgits/gcr-wavefront"
	"github.com/valyala/fasthttp"
)

const (
	servicename = "user"
)

var (
	db datastore.Manager
)

// CORSHandler sets CORS headers for the preflight request
func CORSHandler(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.Add("Access-Control-Allow-Credentials", "true")
	ctx.Response.Header.Add("Access-Control-Allow-Headers", "Authorization")
	ctx.Response.Header.Add("Access-Control-Allow-Methods", "GET, POST")
	ctx.Response.Header.Add("Access-Control-Allow-Origin", "*")
	ctx.Response.Header.Add("Access-Control-Max-Age", "3600")
	ctx.Response.SetStatusCode(http.StatusNoContent)
}

// ErrorHandler takes the activity where the error occured and the error object and sends a message to sentry.
func ErrorHandler(ctx *fasthttp.RequestCtx, function string, method string, err error) {
	sentry.CaptureException(fmt.Errorf("error in %s::%s %s", function, method, err.Error()))
	ctx.SetStatusCode(http.StatusBadRequest)
	ctx.SetBodyString(err.Error())
}

func main() {
	// Get the version or set a default to "dev"
	version := os.Getenv("VERSION")
	if version == "" {
		version = "dev"
	}

	// Get the Wavefront server URL or set it to debug
	wfServer := os.Getenv("WAVEFRONT_URL")
	if wfServer == "" {
		wfServer = gcrwavefront.DebugServerName
	}

	// Get the server port or set it to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Get the service name
	service := os.Getenv("K_SERVICE")
	if service == "" {
		service = servicename
	}

	// Initialize a connection to Sentry to capture errors and traces
	if err := sentry.Init(sentry.ClientOptions{
		Dsn: os.Getenv("SENTRY_DSN"),
		Transport: &sentry.HTTPSyncTransport{
			Timeout: time.Second * 3,
		},
		ServerName:  service,
		Release:     version,
		Environment: os.Getenv("STAGE"),
	}); err != nil {
		log.Fatalf("error configuring sentry: %s", err.Error())
	}

	// Create an instance of sentryfasthttp
	sentryHandler := sentryfasthttp.New(sentryfasthttp.Options{})

	// Configure the Wavefront wrapper
	cfg := gcrwavefront.WavefrontConfig{
		Server:        wfServer,
		Token:         os.Getenv("WAVEFRONT_TOKEN"),
		BatchSize:     10000,
		MaxBufferSize: 50000,
		FlushInterval: 1,
		Source:        "acmeserverless",
		MetricPrefix:  fmt.Sprintf("acmeserverless.gcr.%s", servicename),
		PointTags:     make(map[string]string),
	}

	if err := cfg.ConfigureSender(); err != nil {
		log.Fatalf("error configuring wavefront: %s", err.Error())
	}

	// Wrap the sentryHandler with the Wavefront middleware to make sure all events
	// are sent to sentry before sending data to Wavefront
	router := router.New()
	router.GlobalOPTIONS = CORSHandler

	// Add routes to the router
	router.GET("/users", cfg.WrapFastHTTPRequest(sentryHandler.Handle(GetAllUsers)))
	router.GET("/users/{id}", cfg.WrapFastHTTPRequest(sentryHandler.Handle(GetUserDetails)))
	router.POST("/register", cfg.WrapFastHTTPRequest(sentryHandler.Handle(RegisterUser)))
	router.POST("/login", cfg.WrapFastHTTPRequest(sentryHandler.Handle(Login)))
	router.POST("/refresh-token", cfg.WrapFastHTTPRequest(sentryHandler.Handle(RefreshJWTToken)))
	router.POST("/verify-token", cfg.WrapFastHTTPRequest(sentryHandler.Handle(VerifyJWTToken)))

	// Create an instance of the datastore manager
	db = mongodb.New()

	// Start the server
	log.Printf("successfully started %s server", servicename)
	log.Fatal(fasthttp.ListenAndServe(fmt.Sprintf(":%s", port), router.Handler))
}
