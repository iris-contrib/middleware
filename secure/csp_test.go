package secure

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/kataras/iris/v12"
)

// cspHandler writes the nonce out as the response body.
var cspHandler = func(ctx iris.Context) {
	ctx.WriteString(CSPNonce(ctx))
}

func TestCSPNonce(t *testing.T) {
	csp := "default-src 'self' $NONCE; script-src 'strict-dynamic' $NONCE"
	cases := []struct {
		options Options
		headers []string
	}{
		{Options{ContentSecurityPolicy: csp}, []string{"Content-Security-Policy"}},
		{Options{ContentSecurityPolicyReportOnly: csp}, []string{"Content-Security-Policy-Report-Only"}},
		{Options{ContentSecurityPolicy: csp, ContentSecurityPolicyReportOnly: csp},
			[]string{"Content-Security-Policy", "Content-Security-Policy-Report-Only"}},
	}

	for _, c := range cases {
		s := New(c.options)
		app := iris.New()
		app.Use(s.Handler)
		app.Get("/foo", cspHandler)
		if err := app.Build(); err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/foo", nil)
		app.ServeHTTP(res, req)
		expect(t, res.Code, http.StatusOK)

		for _, header := range c.headers {
			csp := res.Header().Get(header)
			expect(t, strings.Count(csp, "'nonce-"), 2)

			nonce := strings.Split(strings.Split(csp, "'")[3], "-")[1]
			// Test that the context has the CSP nonce.
			expect(t, res.Body.String(), nonce)

			_, err := base64.RawStdEncoding.DecodeString(nonce)
			expect(t, err, nil)

			expect(t, csp, fmt.Sprintf("default-src 'self' 'nonce-%[1]s'; script-src 'strict-dynamic' 'nonce-%[1]s'", nonce))
		}
	}
}

func expect(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Errorf("Expected [%v] (type %v) - Got [%v] (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}
