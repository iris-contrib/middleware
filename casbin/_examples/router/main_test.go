package main

import (
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

func TestCasbinRouter(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app, httptest.Debug(false))

	type ttcasbin struct {
		username string
		password string
		path     string
		method   string
		status   int
	}

	tt := []ttcasbin{
		{"alice", "alicepass", "/dataset1/resource1", "GET", 200},
		{"alice", "alicepass", "/dataset1/resource1", "POST", 200},
		{"alice", "alicepass", "/dataset1/resource2", "GET", 200},
		{"alice", "alicepass", "/dataset1/resource2", "POST", 403},

		{"bob", "bobpass", "/dataset2/resource1", "GET", 200},
		{"bob", "bobpass", "/dataset2/resource1", "POST", 200},
		{"bob", "bobpass", "/dataset2/resource1", "DELETE", 200},
		{"bob", "bobpass", "/dataset2/resource2", "GET", 200},
		{"bob", "bobpass", "/dataset2/resource2", "POST", 403},
		{"bob", "bobpass", "/dataset2/resource2", "DELETE", 403},

		{"bob", "bobpass", "/dataset2/folder1/item1", "GET", 403},
		{"bob", "bobpass", "/dataset2/folder1/item1", "POST", 200},
		{"bob", "bobpass", "/dataset2/folder1/item1", "DELETE", 403},
		{"bob", "bobpass", "/dataset2/folder1/item2", "GET", 403},
		{"bob", "bobpass", "/dataset2/folder1/item2", "POST", 200},
		{"bob", "bobpass", "/dataset2/folder1/item2", "DELETE", 403},
	}

	for _, tt := range tt {
		check(e, tt.method, tt.path, tt.username, tt.password, tt.status)
	}
}

func check(e *httptest.Expect, method, path, username, password string, status int) {
	e.Request(method, path).WithBasicAuth(username, password).Expect().Status(status)
}
