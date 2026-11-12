# How to Use Iris and PostgreSQL for Web Development

This walkthrough builds a small REST API for customers with [Iris](https://www.iris-go.com/) v14 and PostgreSQL, using the PG middleware of this repository. The complete program is [main.go](main.go).

## What is PG middleware?

PG middleware is a package for the Iris web framework that gives request handlers typed access to a PostgreSQL database:

- It supports PostgreSQL 9.5 and above.
- It uses the [pg](https://github.com/kataras/pg) package and the [pgx](https://github.com/jackc/pgx) driver under the hood.
- It can run each request in a transaction, create and check the schema, and trace queries.
- It registers custom types and table models through a schema object.
- It provides a generic repository for common CRUD operations.

## How to install PG middleware?

```sh
go get github.com/iris-contrib/middleware/pg@main
```

## How to use PG middleware?

1. Import the packages:

```go
import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"

	"github.com/iris-contrib/middleware/pg"
)
```

2. Define your database table models as structs with `json` and `pg` tags, and the body your API accepts:

```go
// The Customer database table model.
type Customer struct {
	ID   string `json:"id" pg:"type=uuid,primary"`
	Name string `json:"name" pg:"type=varchar(255)"`
}

// The body of the create and update requests.
type customerInput struct {
	Name string `json:"name"`
}
```

The `json` tag defines how the struct fields are encoded or decoded as JSON. The `pg` tag defines how the struct fields are mapped to the database table columns: column type, constraints, indexes and other options. Read more at the [pg](https://github.com/kataras/pg) repository.

3. Create a schema object and register your models:

```go
schema := pg.NewSchema()
schema.MustRegister("customers", Customer{})
```

The schema object stores the metadata of your database tables and models. Register each model with `MustRegister`: the first argument is the table name, the second is the model type.

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
	Transactional: true,
	Trace:         true,
	CreateSchema:  true,
	CheckSchema:   true,
}

p := pg.New(schema, opts)
```

The connection fields (host, port, user, password, dbname, schema, sslmode) establish the connection. The other fields control the middleware: `Transactional` runs each request in a database transaction, `Trace` logs each executed query, `CreateSchema` creates the tables that don't exist and `CheckSchema` reports missing tables and columns.

There is also an optional `ErrorHandler` field. It receives the errors of the middleware itself, such as a transaction that cannot start or commit. Return `true` from it to mark the error handled, or `false` to fall through to `pg.DefaultErrorHandler`. This example does not set it.

`New` connects to the database and panics when it cannot.

5. Attach the middleware handler to your Iris app or routes:

```go
app := iris.New()

postgresMiddleware := p.Handler()

customerAPI := app.Router("/api/customer", postgresMiddleware)
customerAPI.Post("/", createCustomer)
customerAPI.Get("/{id:uuid}", getCustomer)
customerAPI.Put("/{id:uuid}", updateCustomer)
customerAPI.Delete("/{id:uuid}", deleteCustomer)
```

The handler stores the database on the request Context and calls the next handler. Attach it with `Use`, or pass it to `Router` as done here for the customer API group.

6. Use the `pg.DB` or `pg.Repository` package-level functions to reach the database or a typed repository in your handlers:

```go
// fail answers a database error and marks the request as failed.
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

	// Get the current database instance through the pg.DB package-level function.
	// db := pg.DB(ctx)
	// [Work with db instance...]
	// OR, initialize a new repository of Customer type and work with it (type-safety).
	customers := pg.Repository[Customer](ctx)

	// Insert a new Customer.
	customer := Customer{
		Name: payload.Name,
	}
	err := customers.InsertSingle(ctx, customer, &customer.ID)
	if err != nil {
		fail(ctx, err)
		return
	}

	// Display the result ID.
	ctx.StatusCode(iris.StatusCreated)
	ctx.JSON(iris.Map{"id": customer.ID})
}

func getCustomer(ctx iris.Context) {
	// Get the id from the path parameter.
	id := ctx.Params().GetString("id")

	// Get the repository of Customer type through the pg.Repository package-level function.
	customers := pg.Repository[Customer](ctx)

	// Get the customer by the id.
	customer, err := customers.SelectByID(ctx, id)
	if err != nil {
		fail(ctx, err) // 404 when there is no such customer.
		return
	}

	// Display the retrieved Customer.
	ctx.JSON(customer)
}

func updateCustomer(ctx iris.Context) {
	// Get the id from the path parameter.
	id := ctx.Params().GetString("id")

	payload, ok := ctx.BindJSON[customerInput]()
	if !ok {
		return
	}

	// Get the repository of Customer type through the pg.Repository package-level function.
	customers := pg.Repository[Customer](ctx)

	// Update the customer by the id and name.
	customer := Customer{
		ID:   id,
		Name: payload.Name,
	}

	n, err := customers.UpdateOnlyColumns(ctx, []string{"name"}, customer)
	// OR customers.Update(ctx, customer)
	if err != nil {
		fail(ctx, err)
		return
	}

	if n == 0 {
		errors.NotFound.Message(ctx, "customer not found")
		return
	}

	// Display a success message.
	ctx.JSON(iris.Map{"message": "Customer updated successfully"})
}

func deleteCustomer(ctx iris.Context) {
	// Get the id from the path parameter.
	id := ctx.Params().GetString("id")

	// Get the repository of Customer type through the pg.Repository package-level function.
	customers := pg.Repository[Customer](ctx)

	// Delete the customer by the id.
	deleted, err := customers.DeleteByID(ctx, id)
	if err != nil {
		fail(ctx, err)
		return
	}

	if !deleted {
		errors.NotFound.Message(ctx, "customer not found")
		return
	}

	// Display a success message.
	ctx.JSON(iris.Map{"message": "Customer deleted successfully"})
}
```

The `pg.DB` function returns the database of the current request, which is the transaction when `Transactional` is on; use it for any operation the [pg](https://github.com/kataras/pg) package offers. The `pg.Repository` function returns a generic repository for the given model type, with typed methods for common CRUD operations.

Two things happen in `fail`. `errors.HandleError` writes the response through the framework's `rest/errors` package: 404 when there are no rows, 409 for a duplicate key, 400 for a value PostgreSQL cannot parse and a generic 500 for anything else. The database's own message stays in the log. `ctx.SetErr` stores the error on the Context, which is what tells the middleware to roll the transaction back instead of committing it.

## How to run the example?

1. Clone the [repository](https://github.com/iris-contrib/middleware) and go to the `pg/_examples/basic` folder.
2. Start a PostgreSQL server and create a database named `test_db`.
3. Run the program with `go run .`.
4. Use a tool like [curl](https://curl.se/) to test the API endpoints:

```sh
# Create a new customer
curl -X POST -H "Content-Type: application/json" -d '{"name":"Alice"}' http://localhost:8080/api/customer

# Get a customer by id
curl -X GET http://localhost:8080/api/customer/1f8c9a7c-6b7c-4f0e-8a1d-9f2a9c3b7b8e

# Update a customer by id
curl -X PUT -H "Content-Type: application/json" -d '{"name":"Bob"}' http://localhost:8080/api/customer/1f8c9a7c-6b7c-4f0e-8a1d-9f2a9c3b7b8e

# Delete a customer by id
curl -X DELETE http://localhost:8080/api/customer/1f8c9a7c-6b7c-4f0e-8a1d-9f2a9c3b7b8e
```

## Earlier versions of this article

The first version of this walkthrough, written for Iris v12, was published at:

1. https://dev.to/kataras/how-to-use-iris-and-postgresql-for-web-development-3kka
	- https://twitter.com/TheGoDev/status/1676701488544874498
	- https://twitter.com/TheDatabaseDev/status/1676748294595174401
2. https://medium.com/@kataras/how-to-use-iris-and-postgresql-for-web-development-e8c46d72f1e6
3. https://www.linkedin.com/pulse/how-use-iris-postgresql-web-development-gerasimos-maropoulos

The code there uses the v12 API. This file is the current one.

If you have any feedback or questions, please open an issue or a pull request on [GitHub](https://github.com/iris-contrib/middleware/tree/main/pg).
