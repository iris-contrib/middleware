package main

import (
	"regexp"
	"testing"

	"github.com/iris-contrib/middleware/csrf"

	"github.com/kataras/iris/v14/httptest"
)

var fieldValue = regexp.MustCompile(`<input type="hidden" name="csrf\.token" value="([^"]+)">`)

func TestSignup(t *testing.T) {
	e := httptest.New(t, newApp())

	// The form is rendered with the hidden field and the cookie is issued.
	r := e.GET("/user/signup").Expect().Status(httptest.StatusOK)
	cookie := r.Cookie(csrf.DefaultCookieName).Value().NotEmpty().Raw()

	match := fieldValue.FindStringSubmatch(r.Body().Raw())
	if match == nil {
		t.Fatalf("expected the rendered form to contain the hidden field, got:\n%s", r.Body().Raw())
	}
	token := match[1]

	e.POST("/user/signup").WithCookie(csrf.DefaultCookieName, cookie).
		WithFormField(csrf.DefaultFieldName, token).
		Expect().Status(httptest.StatusOK).Body().IsEqual("You're welcome mate!")

	// Without the token the request is refused.
	e.POST("/user/signup").WithCookie(csrf.DefaultCookieName, cookie).
		Expect().Status(httptest.StatusForbidden)

	// The route outside the protected router is not checked.
	e.POST("/user/unprotected").Expect().Status(httptest.StatusOK).
		Body().IsEqual("Hey, I am open to CSRF attacks!")
}
