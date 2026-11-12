package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/types"

	"github.com/iris-contrib/middleware/pg"
)

// The Customer database table model.
type Customer struct {
	ID        string        `json:"id" pg:"type=uuid,primary"`
	CreatedAt types.ISO8601 `pg:"type=timestamp,default=clock_timestamp()" json:"created_at,omitempty"`
	UpdatedAt types.ISO8601 `pg:"type=timestamp,default=clock_timestamp()" json:"updated_at,omitempty"`

	Name string `json:"name" pg:"type=varchar(255)"`
}

func newPG() *pg.PG {
	schema := pg.NewSchema()
	schema.MustRegister("customers", Customer{})

	opts := pg.Options{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "admin!123",
		DBName:   "test_db",
		Schema:   "public",
		SSLMode:  "disable",
		// Transactional is left out: it applies to the handlers registered
		// behind pgMiddleware.Handler(). The controller talks to the database
		// directly and each of its routes runs a single statement.
		Trace:        true, // or false to production to disable query logging.
		CreateSchema: true, // true to create the schema if it doesn't exist.
		CheckSchema:  true, // true to check the schema for missing tables and columns.
		// ErrorHandler is optional. The controller answers a database error
		// through it; without it pg.DefaultErrorHandler does: 404 when there
		// are no rows, 409 for a duplicate key and so on.
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
	app.Router("/api/customer", customerController)

	// GET /api/customer/schema
	// GET /api/customer/{id}
	// POST /api/customer
	// PUT /api/customer
	// DELETE /api/customer/{id}
	app.Listen(":8080")
}
