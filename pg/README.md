# PG Middleware

PG middleware is a package for [Iris](https://iris-go.com/) web framework that provides easy and type-safe access to PostgreSQL database.

## Features

- Supports PostgreSQL 9.5 and above.
- Uses [pg](https://github.com/kataras/pg) package and [pgx](https://github.com/jackc/pgx) driver under the hood.
- Supports transactions, schema creation and validation, query tracing and error handling.
- Allows registering custom types and table models using a schema object.
- Provides a generic repository interface for common CRUD operations.

## Installation

To install PG middleware, use the following command:

```sh
go get github.com/iris-contrib/middleware/pg@master
```

## Usage

To use PG middleware, you need to:

1. Import the package in your code:

```go
import (
	"github.com/kataras/iris/v12"

	"github.com/iris-contrib/middleware/pg"
)
```

2. Define your database table models as structs with `json` and `pg` tags:

```go
// The Customer database table model.
type Customer struct {
	ID   string `json:"id" pg:"type=uuid,primary"`
	Name string `json:"name" pg:"type=varchar(255)"`
}
```

3. Create a schema object and register your models:

```go
schema := pg.NewSchema()
schema.MustRegister("customers", Customer{})
```

4. Create a PG middleware instance with the schema and database options:

```go
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
	ErrorHandler: func(ctx iris.Context, err error) {
		ctx.StopWithError(iris.StatusInternalServerError, err)
	},
}

p := pg.New(schema, opts)
```

5. Attach the middleware handler to your Iris app or routes:

```go
app := iris.New()

postgresMiddleware := newPostgresMiddleware()

{
	customerAPI := app.Party("/api/customer", postgresMiddleware)
	customerAPI.Post("/", createCustomer)
	customerAPI.Get("/{id:uuid}", getCustomer)
}
```

6. Use the `pg.DB` or `pg.Repository` package-level functions to access the database instance or the repository interface in your handlers:

```go
func createCustomer(ctx iris.Context) {
	var payload = struct {
		Name string `json:"name"`
	}{}
	err := ctx.ReadJSON(&payload)
	if err != nil {
		ctx.StopWithError(iris.StatusBadRequest, err)
		return
	}

	// Get the current database instance through pg.DB middleware package-level function.
	// db := pg.DB(ctx)
	// [Work with db instance...]
	// OR, initialize a new repository of Customer type and work with it (type-safety).
	customers := pg.Repository[Customer](ctx)

	// Insert a new Customer.
	customer := Customer{
		Name: payload.Name,
	}
	err = customers.InsertSingle(ctx, customer, &customer.ID)
	if err != nil {
		ctx.StopWithError(iris.StatusInternalServerError, err)
		return
	}

	// Display the result ID.
	ctx.StatusCode(iris.StatusCreated)
	ctx.JSON(iris.Map{"id": customer.ID})
}

func getCustomer(ctx iris.Context) {
	// Get the id from the path parameter.
	id := ctx.Params().Get("id")

	// Get the repository of Customer type through pg.Repository middleware package-level function.
	customers := pg.Repository[Customer](ctx)

	// Get the customer by the id.
	customer, err := customers.SelectByID(ctx, id)
	if err != nil {
		if pg.IsErrNoRows(err) {
			ctx.StopWithStatus(iris.StatusNotFound)
		} else {
			ctx.StopWithError(iris.StatusInternalServerError, err)
		}

		return
	}

	// Display the retrieved Customer.
	ctx.JSON(customer)
}
```

## Examples

You can find more examples of using PG middleware in the [examples](./examples) folder.

## License

PG middleware is licensed under the [MIT License](./LICENSE).