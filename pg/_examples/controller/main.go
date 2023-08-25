package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/x/jsonx"

	"github.com/iris-contrib/middleware/pg"
)

// The Customer database table model.
type Customer struct {
	ID        string        `json:"id" pg:"type=uuid,primary"`
	CreatedAt jsonx.ISO8601 `pg:"type=timestamp,default=clock_timestamp()" json:"created_at,omitempty"`
	UpdatedAt jsonx.ISO8601 `pg:"type=timestamp,default=clock_timestamp()" json:"updated_at,omitempty"`

	Name string `json:"name" pg:"type=varchar(255)"`
}

func newPG() *pg.PG {
	schema := pg.NewSchema()
	schema.MustRegister("customers", Customer{})

	opts := pg.Options{
		Host:          "localhost",
		Port:          5432,
		User:          "postgres",
		Password:      "admin!123",
		DBName:        "test_db",
		Schema:        "public",
		SSLMode:       "disable",
		Transactional: true, // or false to disable the transactional feature.
		Trace:         true, // or false to production to disable query logging.
		CreateSchema:  true, // true to create the schema if it doesn't exist.
		CheckSchema:   true, // true to check the schema for missing tables and columns.
		ErrorHandler: func(ctx iris.Context, err error) bool {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return true
		},
	}

	p := pg.New(schema, opts)
	return p
}

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")

	pgMiddleware := newPG()
	defer pgMiddleware.Close()

	customerController := pg.NewEntityController[Customer](pgMiddleware) // .WithoutSchemaRoute()
	app.PartyConfigure("/api/customer", customerController)

	// GET /api/customer/schema
	// GET /api/customer/{id}
	// POST /api/customer
	// PUT /api/customer
	// DELETE /api/customer/{id}
	app.Listen(":8080")
}
