# pg

PostgreSQL access for [Iris](https://github.com/kataras/iris) v14 through [kataras/pg](https://github.com/kataras/pg), which is built on the [pgx](https://github.com/jackc/pgx) driver.

The module gives you three things:

- a handler that puts the database, or a transaction on it, on the request Context
- `pg.DB(ctx)` and the generic `pg.Repository[T](ctx)` to reach it from your handlers
- `EntityController[T]`, a create, read, update and delete API for one table

## Install

```sh
go get github.com/iris-contrib/middleware/pg@main
```

```go
import "github.com/iris-contrib/middleware/pg"
```

The package is named `pg`, like `github.com/kataras/pg`. For `pg.NewSchema`, `pg.ErrNoRows` and `pg.IsErrNoRows` this module re-exports what you need. Import `github.com/kataras/pg` under another name when you need more of it.

## Example

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"

	"github.com/iris-contrib/middleware/pg"
)

// The Customer database table model.
type Customer struct {
	ID   string `json:"id" pg:"type=uuid,primary"`
	Name string `json:"name" pg:"type=varchar(255)"`
}

type customerInput struct {
	Name string `json:"name"`
}

func main() {
	schema := pg.NewSchema()
	schema.MustRegister("customers", Customer{})

	// New connects to the database and panics when it cannot.
	p := pg.New(schema, pg.Options{
		Host:          "localhost",
		Port:          5432,
		User:          "postgres",
		Password:      "admin!123",
		DBName:        "test_db",
		Schema:        "public",
		SSLMode:       "disable",
		Transactional: true, // run each request in a transaction.
		Trace:         true, // log every query. Turn it off in production.
		CreateSchema:  true, // create the tables that do not exist.
		CheckSchema:   true, // report missing tables and columns.
	})
	defer p.Close()

	app := iris.New()

	customerAPI := app.Router("/api/customer", p.Handler())
	customerAPI.Post("/", createCustomer)
	customerAPI.Get("/{id:uuid}", getCustomer)

	app.Listen(":8080")
}

// fail answers a database error and marks the request as failed.
// errors.HandleError writes the response (404 for no rows, 409 for a
// duplicate key and so on). ctx.SetErr is what rolls the transaction back.
func fail(ctx iris.Context, err error) {
	ctx.SetErr(iris.PrivateError(err))
	errors.HandleError(ctx, err)
}

func createCustomer(ctx iris.Context) {
	// BindJSON answers 400 by itself when the body is not valid JSON.
	payload, ok := ctx.BindJSON[customerInput]()
	if !ok {
		return
	}

	// The repository of the Customer type on this request's transaction.
	// pg.DB(ctx) returns the *pg.DB itself.
	customers := pg.Repository[Customer](ctx)

	customer := Customer{Name: payload.Name}
	if err := customers.InsertSingle(ctx, customer, &customer.ID); err != nil {
		fail(ctx, err)
		return
	}

	ctx.StatusCode(iris.StatusCreated)
	ctx.JSON(iris.Map{"id": customer.ID})
}

func getCustomer(ctx iris.Context) {
	customers := pg.Repository[Customer](ctx)

	customer, err := customers.SelectByID(ctx, ctx.Params().GetString("id"))
	if err != nil {
		fail(ctx, err)
		return
	}

	ctx.JSON(customer)
}
```

## Options

| Field | Description |
| --- | --- |
| `Host`, `Port`, `User`, `Password`, `DBName`, `SSLMode` | The connection. Use `SSLMode: "verify-full"` outside your own machine. |
| `Schema` | The PostgreSQL schema, sent as the connection's `search_path`. |
| `Trace` | Logs every query through the default [golog](https://github.com/kataras/golog) logger. |
| `Transactional` | Runs each request that goes through `Handler()` in a transaction. See below. |
| `CreateSchema` | Creates the registered tables that do not exist, when the middleware is created. |
| `CheckSchema` | Checks the database for missing tables and columns, when the middleware is created. |
| `ErrorHandler` | `func(ctx iris.Context, err error) bool`. Receives the errors of the middleware and of the entity controller. Return false to let `DefaultErrorHandler` answer. |

The fields have `yaml` tags, so the options can be part of your configuration file.

## API

| Name | Description |
| --- | --- |
| `New(schema *pg.Schema, opts Options) *PG` | Opens the database with a timeout of 10 seconds. Panics when it cannot connect, or when `CreateSchema` or `CheckSchema` fail. |
| `NewFromDB(db *pg.DB, opts Options) *PG` | Uses a database you opened yourself. The connection fields and `Trace` must be left unset. |
| `(*PG).Handler() iris.Handler` | The middleware. |
| `(*PG).GetDB() *pg.DB` | The database, outside a request. |
| `(*PG).Close()` | Closes the connection pool. |
| `DB(ctx) *pg.DB` | The database of the current request: the transaction when there is one. Nil when the handler did not run. |
| `Repository[T](ctx) *pg.Repository[T]` | A repository of `T` on `DB(ctx)`. Nil when the handler did not run. |
| `DefaultErrorHandler` | The error handler used when `Options.ErrorHandler` is nil or returns false. |
| `NewSchema`, `ErrNoRows`, `IsErrNoRows` | Re-exported from `github.com/kataras/pg`. |

## Transactions

With `Transactional: true` the handler begins a transaction before the next handlers run and stores it on the Context. When they return:

| What happened | Result |
| --- | --- |
| No error is stored on the Context | commit |
| A handler stored an error (`ctx.StopWithError`, `ctx.StopWithPlainError`, `ctx.SetErr`) | rollback |
| A handler stored `pg.ErrIntentionalRollback` | rollback, and the response the handler wrote stays as it is |
| A handler panicked | rollback, and the panic continues to your recovery middleware |
| The transaction could not begin or commit | the error handler answers |

The functions of the `rest/errors` package (`errors.HandleError`, `errors.NotFound.Message` and the rest) write a response but do not store an error. A handler that answers through them after it has written to the database must also call `ctx.SetErr(err)` to roll back, as `fail` does in the example. Without it the transaction is committed. (When the failure was a statement that PostgreSQL rejected, the transaction is already aborted and the commit rolls it back anyway.)

## Errors

`DefaultErrorHandler` answers through the framework's [rest/errors](https://github.com/kataras/iris/tree/main/rest/errors) package, with the mapping that `rest/errors/pgerrors` registers. Importing this module registers that mapping, so `errors.HandleError(ctx, err)` in your own handlers uses it too.

| Error | Response |
| --- | --- |
| `pg.ErrNoRows` | 404 |
| A duplicate key (SQLSTATE 23505) | 409 |
| A foreign key violation (SQLSTATE 23503) | 400, naming the constraint |
| A value PostgreSQL could not parse (`invalid input syntax`) | 400 |
| A value that is not a number (`strconv.ErrSyntax`) | 400 |
| Anything else | 500 with a generic message |

The database's own message names tables, columns and the offending value. It is logged and not sent to the client.

## EntityController

`NewEntityController[T](p)` returns a controller that serves one registered table. Pass it to a router:

```go
package main

import (
	"github.com/kataras/iris/v14"

	"github.com/iris-contrib/middleware/pg"
)

type Customer struct {
	ID   string `json:"id" pg:"type=uuid,primary"`
	Name string `json:"name" pg:"type=varchar(255)"`
}

func main() {
	schema := pg.NewSchema()
	schema.MustRegister("customers", Customer{})

	p := pg.New(schema, pg.Options{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "admin!123",
		DBName:   "test_db",
		Schema:   "public",
		SSLMode:  "disable",
	})
	defer p.Close()

	app := iris.New()
	app.Router("/api/customer", pg.NewEntityController[Customer](p))

	app.Listen(":8080")
}
```

| Route | Description |
| --- | --- |
| `GET /schema` | The table's columns as JSON. Remove it with `WithoutSchemaRoute()`. |
| `POST /` | Inserts the entity of the JSON body. Answers 201 with `{"id": ...}`. |
| `PUT /` | Updates the entity of the JSON body by its primary key. `?columns=name,email` updates only those columns. Answers 204, or 404 when no row matched. |
| `GET /{id}` | Answers the entity, or 404. |
| `DELETE /{id}` | Answers 204, or 404 when no row matched. |

The `{id}` path parameter is typed after the primary key column: `uuid`, `int` or `string`. An id of the wrong type matches no route and answers 404.

| Field | Description |
| --- | --- |
| `GetID func(ctx iris.Context) any` | Returns the id for `GET` and `DELETE`. Defaults to the path parameter. |
| `ErrorHandler func(ctx iris.Context, err error) bool` | Answers a database error. Defaults to the middleware's. |
| `AfterPayloadRead func(ctx iris.Context, payload T) (T, bool)` | Runs after the body is read. Return false, after writing a response, to stop. |

The body is read with `ctx.BindJSON`: invalid JSON answers 400, and so does an entity whose `Validate() error` method returns an error.

Two limits to know about:

- The controller talks to the database directly and each route runs one statement. `Options.Transactional` does not apply to it, with or without `p.Handler()` in front.
- A table without a primary key panics in `NewEntityController`.

## Examples

- [_examples/basic](_examples/basic): the handler and repositories, with a walkthrough
- [_examples/controller](_examples/controller/main.go): the entity controller

## Tests

The tests run against a fake PostgreSQL server inside the test process: a connection is one end of a `net.Pipe`, and the fake answers the startup handshake and the simple query protocol. They cover the transaction flow, the error mapping and every route of the controller. What a real PostgreSQL does with the statements is not tested here: `New`, `CreateSchema`, `CheckSchema`, query tracing and the extended query protocol need a live server.

## License

[MIT](LICENSE)
