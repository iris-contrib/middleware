package main

import (
	"testing"

	"github.com/kataras/iris/v14/httptest"
)

func TestProtected(t *testing.T) {
	e := httptest.New(t, newApp())

	e.GET("/protected").Expect().Status(httptest.StatusUnauthorized)

	token := e.GET("/").Expect().Status(httptest.StatusOK).Body().NotEmpty().Raw()

	e.GET("/protected").WithHeader("Authorization", "Bearer "+token).
		Expect().Status(httptest.StatusOK).Body().Contains("foo=bar")

	// The token is not read from the URL.
	e.GET("/protected").WithQuery("token", token).Expect().Status(httptest.StatusUnauthorized)
}
