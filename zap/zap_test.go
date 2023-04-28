package iriszap

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kataras/iris/v12"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func buildDummyLogger() (*zap.Logger, *observer.ObservedLogs) {
	core, obs := observer.New(zap.InfoLevel)
	logger := zap.New(core)
	return logger, obs
}

func timestampLocationCheck(t *testing.T, timestampStr string, location *time.Location) error {
	timestamp, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		return err
	}
	if timestamp.Location() != location {
		return fmt.Errorf("timestamp should be utc but %v", timestamp.Location())
	}

	return nil
}

func TestNew(t *testing.T) {
	app := iris.New()

	utcLogger, utcLoggerObserved := buildDummyLogger()
	app.Use(New(utcLogger, time.RFC3339, true))

	localLogger, localLoggerObserved := buildDummyLogger()
	app.Use(New(localLogger, time.RFC3339, false))

	app.Get("/test", func(ctx iris.Context) {
		ctx.StatusCode(iris.StatusNoContent)
		ctx.JSON(nil)
	})

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	res1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/test", nil)
	app.ServeHTTP(res1, req1)

	if len(utcLoggerObserved.All()) != 1 {
		t.Fatalf("Log should be 1 line but there're %d", len(utcLoggerObserved.All()))
	}

	logLine := utcLoggerObserved.All()[0]
	pathStr := logLine.Context[2].String
	if pathStr != "/test" {
		t.Fatalf("logged path should be /test but %s", pathStr)
	}

	err := timestampLocationCheck(t, logLine.Context[7].String, time.UTC)
	if err != nil {
		t.Fatal(err)
	}

	if len(localLoggerObserved.All()) != 1 {
		t.Fatalf("Log should be 1 line but there're %d", len(utcLoggerObserved.All()))
	}

	logLine = localLoggerObserved.All()[0]
	pathStr = logLine.Context[2].String
	if pathStr != "/test" {
		t.Fatalf("logged path should be /test but %s", pathStr)
	}
}

func TestNewWithConfig(t *testing.T) {
	app := iris.New()

	utcLogger, utcLoggerObserved := buildDummyLogger()
	app.Use(NewWithConfig(utcLogger, &Config{
		TimeFormat: time.RFC3339,
		UTC:        true,
		SkipPaths:  []string{"/no_log"},
	}))

	app.Get("/test", func(ctx iris.Context) {
		ctx.StatusCode(iris.StatusNoContent)
		ctx.JSON(nil)
	})

	app.Get("/no_log", func(ctx iris.Context) {
		ctx.StatusCode(iris.StatusNoContent)
		ctx.JSON(nil)
	})

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	res1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/test", nil)
	app.ServeHTTP(res1, req1)

	res2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/no_log", nil)
	app.ServeHTTP(res2, req2)

	if res2.Code != iris.StatusNoContent {
		t.Fatalf("request /no_log is failed (%d)", res2.Code)
	}

	if len(utcLoggerObserved.All()) != 1 {
		t.Fatalf("Log should be 1 line but there're %d", len(utcLoggerObserved.All()))
	}

	logLine := utcLoggerObserved.All()[0]
	pathStr := logLine.Context[2].String
	if pathStr != "/test" {
		t.Fatalf("logged path should be /test but %s", pathStr)
	}

	err := timestampLocationCheck(t, logLine.Context[7].String, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
}
