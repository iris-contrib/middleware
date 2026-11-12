package expmetric

import (
	"expvar"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

func TestHitsTotal(t *testing.T) {
	app := iris.New()
	// On the root application Use also runs for requests that match no route.
	app.Use(HitsTotal(MetricName("test_hits_total")))
	app.Get("/", func(ctx iris.Context) { ctx.WriteString("ok") })
	app.Get("/debug/vars", Handler())

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK)
	e.GET("/").Expect().Status(httptest.StatusOK)
	e.GET("/missing").Expect().Status(httptest.StatusNotFound)

	if got := expvar.Get("test_hits_total").String(); got != "3" {
		t.Fatalf("expected 3 hits, got %s", got)
	}

	// The expvar handler lists the counter; this request is the fourth hit.
	e.GET("/debug/vars").Expect().Status(httptest.StatusOK).
		JSON().Object().HasValue("test_hits_total", 4)
}

func TestHitsTotalPerPath(t *testing.T) {
	app := iris.New()
	app.UseRelevant(HitsTotalPerPath(MetricName("test_hits_per_path")))
	app.Get("/a", func(ctx iris.Context) {})
	app.Get("/b", func(ctx iris.Context) {})

	e := httptest.New(t, app)
	e.GET("/a").Expect().Status(httptest.StatusOK)
	e.GET("/a").Expect().Status(httptest.StatusOK)
	e.GET("/b").Expect().Status(httptest.StatusOK)
	// UseRelevant runs on matched routes only.
	e.GET("/missing").Expect().Status(httptest.StatusNotFound)

	if got, expected := expvar.Get("test_hits_per_path").String(), `{"/a": 2, "/b": 1}`; got != expected {
		t.Fatalf("expected %s, got %s", expected, got)
	}
}

// A request without a user is served and not counted.
func TestHitsTotalPerUser(t *testing.T) {
	app := iris.New()
	app.Use(func(ctx iris.Context) {
		if id := ctx.Header("X-User-ID"); id != "" {
			if err := ctx.SetUser(iris.Map{"id": id, "email": id + "@example.com"}); err != nil {
				ctx.StopWithError(iris.StatusInternalServerError, err)
				return
			}
		}
		ctx.Next()
	})
	app.Use(HitsTotalPerUserID(MetricName("test_hits_per_user_id")))
	app.Use(HitsTotalPerUserEmail(MetricName("test_hits_per_user_email")))
	app.Get("/", func(ctx iris.Context) { ctx.WriteString("ok") })

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK).Body().IsEqual("ok")
	e.GET("/").WithHeader("X-User-ID", "42").Expect().Status(httptest.StatusOK)
	e.GET("/").WithHeader("X-User-ID", "42").Expect().Status(httptest.StatusOK)

	if got, expected := expvar.Get("test_hits_per_user_id").String(), `{"42": 2}`; got != expected {
		t.Fatalf("expected %s, got %s", expected, got)
	}

	if got, expected := expvar.Get("test_hits_per_user_email").String(), `{"42@example.com": 2}`; got != expected {
		t.Fatalf("expected %s, got %s", expected, got)
	}
}
