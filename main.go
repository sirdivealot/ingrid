package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/sirdivealot/ingrid/api"
)

//go:generate go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen -generate=server -o=./api/server.gen.go -package api ./spec.yaml
//go:generate go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen -generate=types -o=./api/types.gen.go -package api ./spec.yaml

var (
	HOST string = "localhost"
	PORT string = "8080"
	ADDR string
)

func usage() {
	// Print default/would-be used parameters
	fmt.Println(`ingrid [-h]`)
	fmt.Println()
	fmt.Println(`ENV HOST=` + HOST)
	fmt.Println(`ENV PORT=` + PORT)
	fmt.Println()
	fmt.Println(`http server listen on ` + ADDR)
}

func init() {
	if host, ok := os.LookupEnv("HOST"); ok {
		HOST = host
	}
	if port, ok := os.LookupEnv("PORT"); ok {
		PORT = port
	}
	ADDR = net.JoinHostPort(HOST, PORT)

	// Ensure all timestamps are decently formatted
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Value.Kind() == slog.KindTime {
				return slog.String(a.Key, a.Value.Time().Format(time.RFC3339))
			}
			return a
		},
	})))
}

func main() {
	flag.Usage = usage
	flag.Parse()

	slog.Info("initializing")

	srv := setupserver()
	registerhandlers(srv)

	go func() {
		slog.Info("serving http api", slog.String("addr", ADDR))
		if err := srv.Start(ADDR); err != http.ErrServerClosed {
			fatal("serve", "error", err)
		}
	}()

	// No socket reuse -> no graceful reload,
	// But we can at least shutdown gracefully.
	gracefulshutdown(srv)

	slog.Info("shutdown complete, bye")
}

func setupserver() (srv *echo.Echo) {
	slog.Info("setting up echo server")
	srv = echo.New()

	srv.HideBanner = true

	srv.Logger.SetOutput(io.Discard)
	srv.StdLogger.SetOutput(io.Discard)

	srv.Use(echomiddleware.Recover())

	srv.Use(echomiddleware.RequestLoggerWithConfig(echomiddleware.RequestLoggerConfig{
		LogRoutePath:  true,
		LogMethod:     true,
		LogStatus:     true,
		LogError:      true,
		LogLatency:    true,
		LogRequestID:  true,
		HandleError:   true,
		LogValuesFunc: requestlogger,
	}))

	srv.Use(echomiddleware.RateLimiter(echomiddleware.NewRateLimiterMemoryStoreWithConfig(echomiddleware.RateLimiterMemoryStoreConfig{
		Rate:      1,
		Burst:     1,
		ExpiresIn: 24 * time.Hour,
	})))

	srv.Use(echomiddleware.RequestID())

	srv.Use(echomiddleware.BodyLimit("1M"))

	srv.Use(echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		Skipper:      echomiddleware.DefaultSkipper,
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
		MaxAge:       int((30 * 24 * time.Hour).Seconds()),
	}))

	return
}

func requestlogger(c echo.Context, v echomiddleware.RequestLoggerValues) error {
	var (
		lvl   = slog.LevelInfo
		attrs = []slog.Attr{
			slog.String("method", v.Method),
			slog.String("route_path", v.RoutePath),
			slog.Int("status", v.Status),
			slog.String("id", v.RequestID),
			slog.Duration("latency", v.Latency),
		}
	)
	if v.Error != nil {
		attrs = append(attrs, slog.String("err", v.Error.Error()))
		lvl = slog.LevelError
	}
	slog.LogAttrs(context.Background(), lvl, "request",
		attrs...,
	)
	return nil
}

func registerhandlers(srv *echo.Echo) {
	handlers, err := newHandlers()
	if err != nil {
		fatal("handler", "error", err)
	}
	api.RegisterHandlers(srv, handlers)
}

func gracefulshutdown(srv *echo.Echo) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)

	<-c // wait for signal

	slog.Info("shutting down gracefully")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Server.Shutdown(ctx); err != nil {
		slog.Error("shutdown", "error", err)
	}
}

func fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}
