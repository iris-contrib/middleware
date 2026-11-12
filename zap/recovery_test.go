package iriszap

import (
	"errors"
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"

	"go.uber.org/zap/zapcore"
)

func newPanicApp(h iris.Handler) *iris.Application {
	app := iris.New()
	app.Use(h)
	app.Get("/panic", func(ctx iris.Context) {
		panic("An unexpected error happen!")
	})
	app.Get("/ok", func(ctx iris.Context) {
		ctx.WriteString("ok")
	})

	return app
}

func fieldsOf(entry interface{ ContextMap() map[string]interface{} }) map[string]interface{} {
	return entry.ContextMap()
}

func TestRecoveryWithZap(t *testing.T) {
	logger, observed := buildDummyLogger()
	e := httptest.New(t, newPanicApp(RecoveryWithZap(logger, true)))

	e.GET("/ok").Expect().Status(httptest.StatusOK).Body().IsEqual("ok")
	if n := observed.Len(); n != 0 {
		t.Fatalf("expected no entry for a request that did not panic, got %d", n)
	}

	e.GET("/panic").Expect().Status(httptest.StatusInternalServerError).Body().IsEmpty()

	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("expected one entry for the panic, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Level != zapcore.ErrorLevel || entry.Message != "[Recovery from panic]" {
		t.Fatalf("unexpected entry: %s %q", entry.Level, entry.Message)
	}

	fields := fieldsOf(entry)
	if fields["error"] != "An unexpected error happen!" {
		t.Fatalf("expected the panic value in the entry, got %v", fields["error"])
	}

	if request, _ := fields["request"].(string); !strings.HasPrefix(request, "GET /panic HTTP/1.1") {
		t.Fatalf("expected the request line in the entry, got %q", request)
	}

	if stack, _ := fields["stack"].(string); !strings.Contains(stack, "recovery_test.go") {
		t.Fatalf("expected the stack trace in the entry, got %q", stack)
	}
}

func TestRecoveryWithZapWithoutStack(t *testing.T) {
	logger, observed := buildDummyLogger()
	e := httptest.New(t, newPanicApp(RecoveryWithZap(logger, false)))

	e.GET("/panic").Expect().Status(httptest.StatusInternalServerError)

	if _, ok := fieldsOf(observed.All()[0])["stack"]; ok {
		t.Fatal("expected no stack trace in the entry")
	}
}

// The headers that carry credentials are logged without their values,
// and the request the handlers see keeps them.
func TestRecoveryRedactsCredentials(t *testing.T) {
	logger, observed := buildDummyLogger()

	var seen string

	app := iris.New()
	app.Use(RecoveryWithZap(logger, false))
	app.Get("/panic", func(ctx iris.Context) {
		seen = ctx.Header("Authorization")
		panic("boom")
	})

	e := httptest.New(t, app)
	e.GET("/panic").
		WithHeader("Authorization", "Bearer secret-token").
		WithHeader("Cookie", "session=secret-session").
		WithHeader("X-Api-Key", "secret-key").
		WithHeader("X-Trace", "visible").
		Expect().Status(httptest.StatusInternalServerError)

	if seen != "Bearer secret-token" {
		t.Fatalf("expected the handler to read the real header, got %q", seen)
	}

	request, _ := fieldsOf(observed.All()[0])["request"].(string)
	for _, secret := range []string{"secret-token", "secret-session", "secret-key"} {
		if strings.Contains(request, secret) {
			t.Fatalf("the entry contains a credential (%s):\n%s", secret, request)
		}
	}

	for _, expected := range []string{"Authorization: [redacted]", "Cookie: [redacted]", "X-Api-Key: [redacted]", "X-Trace: visible"} {
		if !strings.Contains(request, expected) {
			t.Fatalf("expected the entry to contain %q:\n%s", expected, request)
		}
	}
}

func TestCustomRecoveryWithZap(t *testing.T) {
	logger, observed := buildDummyLogger()

	var recovered interface{}
	h := CustomRecoveryWithZap(logger, false, func(ctx iris.Context, err interface{}) {
		recovered = err
		ctx.StopWithText(iris.StatusServiceUnavailable, "try again later")
	})

	e := httptest.New(t, newPanicApp(h))
	e.GET("/panic").Expect().Status(httptest.StatusServiceUnavailable).Body().IsEqual("try again later")

	if recovered != "An unexpected error happen!" {
		t.Fatalf("expected the panic value in the recovery function, got %v", recovered)
	}

	if observed.Len() != 1 {
		t.Fatalf("expected one entry, got %d", observed.Len())
	}
}

// A request whose handler stored an error on the Context is logged at the
// error level with the error as the message. A request that matched no
// route is logged too, because Use on the root application runs for it.
func TestNewLogsErrorsAndNotFound(t *testing.T) {
	logger, observed := buildDummyLogger()

	app := iris.New()
	app.Use(New(logger, "", false))
	app.Get("/fail", func(ctx iris.Context) {
		ctx.StopWithError(iris.StatusBadGateway, errors.New("upstream is down"))
	})

	e := httptest.New(t, app)
	e.GET("/fail").WithQuery("a", "b").Expect().Status(httptest.StatusBadGateway)
	e.GET("/missing").Expect().Status(httptest.StatusNotFound)

	entries := observed.All()
	if len(entries) != 2 {
		t.Fatalf("expected two entries, got %d", len(entries))
	}

	failed := entries[0]
	if failed.Level != zapcore.ErrorLevel || failed.Message != "upstream is down" {
		t.Fatalf("unexpected entry for the failed request: %s %q", failed.Level, failed.Message)
	}

	fields := fieldsOf(failed)
	if fields["status"] != int64(iris.StatusBadGateway) || fields["path"] != "/fail" || fields["query"] != "a=b" || fields["method"] != "GET" {
		t.Fatalf("unexpected fields for the failed request: %v", fields)
	}

	if _, ok := fields["time"]; ok {
		t.Fatal("expected no time field without a time format")
	}

	missing := entries[1]
	if missing.Level != zapcore.InfoLevel || missing.Message != "/missing" {
		t.Fatalf("unexpected entry for the not found request: %s %q", missing.Level, missing.Message)
	}

	if status := fieldsOf(missing)["status"]; status != int64(iris.StatusNotFound) {
		t.Fatalf("expected the status 404 in the entry, got %v", status)
	}
}

func TestNewWithConfigContext(t *testing.T) {
	logger, observed := buildDummyLogger()

	app := iris.New()
	app.Use(NewWithConfig(logger, &Config{
		Context: func(ctx iris.Context) []zapcore.Field {
			return []zapcore.Field{{Key: "request_id", Type: zapcore.StringType, String: ctx.ResponseHeader("X-Request-Id")}}
		},
	}))
	app.Get("/ping", func(ctx iris.Context) {
		ctx.SetHeader("X-Request-Id", "1234-5678-9012")
		ctx.WriteString("pong")
	})

	httptest.New(t, app).GET("/ping").Expect().Status(httptest.StatusOK)

	if id := fieldsOf(observed.All()[0])["request_id"]; id != "1234-5678-9012" {
		t.Fatalf("expected the request id from the Context func in the entry, got %v", id)
	}
}
