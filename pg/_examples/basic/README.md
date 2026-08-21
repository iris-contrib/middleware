# How to Use Iris and PostgreSQL for Web Development

[Iris](https://www.iris-go.com/) is a fast and lightweight web framework for Go. PostgreSQL is a reliable relational database system with advanced data types and functions. Together they form a solid foundation for building modern web applications.

But how do you connect the two in a simple and type-safe way? How do you perform common database operations without boilerplate, and handle transactions, schema creation, query tracing and errors in one consistent place?

The answer is PG middleware, a package for Iris that provides easy and type-safe access to a PostgreSQL database. In this article we build a small REST API for managing customers with it.

## What is PG middleware?

PG middleware is a package for the Iris web framework that provides easy and type-safe access to a PostgreSQL database:

- It supports PostgreSQL 9.5 and above.
- It uses the [pg](https://github.com/kataras/pg) package and the [pgx](https://github.com/jackc/pgx) driver under the hood.
- It supports transactions, schema creation and validation, query tracing and error handling.
- It registers custom types and table models through a schema object.
- It provides a generic repository interface for common CRUD operations.

## How to install PG middleware?

```sh
go get github.com/iris-contrib/middleware/pg@main
```

## How to use PG middleware?

1. Import the package in your code:

```go
import (
	"github.com/kataras/iris/v14"

	"github.com/iris-contrib/middleware/pg"
)
```

2. Define your database table models as structs with `pg` tags:

```go
// The Customer database table model.
type Customer struct {
	ID   string `json:"id" pg:"type=uuid,primary"`
	Name string `json:"name" pg:"type=varchar(255)"`
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
	ErrorHandler: func(ctx iris.Context, err error) bool {
		ctx.StopWithError(iris.StatusInternalServerError, err)
		return true
	},
}

p := pg.New(schema, opts)
```

The connection fields (host, port, user, password, dbname, schema, sslmode) establish the connection. The feature flags control the middleware behavior: `Transactional` wraps each request handler in a database transaction, `Trace` logs each executed query, `CreateSchema` creates the schema if it doesn't exist and `CheckSchema` reports missing tables and columns. The `ErrorHandler` receives any error that occurs during the middleware execution; return `true` to mark it handled or `false` to fall through to the default error handler.

The `New` function creates the middleware instance from the schema and options.

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

The middleware handler takes an Iris context and calls the next handler in the chain. Attach it with `Use`, or pass it to `Router` as done here for the customer API group.

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

	// Get the current database instance through the pg.DB package-level function.
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
	id := ctx.Params().GetString("id")

	// Get the repository of Customer type through the pg.Repository package-level function.
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

func updateCustomer(ctx iris.Context) {
	// Get the id from the path parameter.
	id := ctx.Params().GetString("id")

	var payload = struct {
		Name string `json:"name"`
	}{}
	err := ctx.ReadJSON(&payload)
	if err != nil {
		ctx.StopWithError(iris.StatusBadRequest, err)
		return
	}

	// Get the repository of Customer type through the pg.Repository package-level function.
	customers := pg.Repository[Customer](ctx)

	// Update the customer by the id and name.
	customer := Customer{
		ID:   id,
		Name: payload.Name,
	}

	_, err = customers.UpdateOnlyColumns(ctx, []string{"name"}, customer)
	// OR customers.Update(ctx, customer)
	if err != nil {
		if pg.IsErrNoRows(err) {
			ctx.StopWithStatus(iris.StatusNotFound)
		} else {
			ctx.StopWithError(iris.StatusInternalServerError, err)
		}

		return
	}

	// Display a success message.
	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(iris.Map{"message": "Customer updated successfully"})
}

func deleteCustomer(ctx iris.Context) {
	// Get the id from the path parameter.
	id := ctx.Params().GetString("id")

	// Get the repository of Customer type through the pg.Repository package-level function.
	customers := pg.Repository[Customer](ctx)

	// Delete the customer by the id.
	_, err := customers.Delete(ctx, Customer{ID: id})
	if err != nil {
		if pg.IsErrNoRows(err) {
			ctx.StopWithStatus(iris.StatusNotFound)
		} else {
			ctx.StopWithError(iris.StatusInternalServerError, err)
		}

		return
	}

	// Display a success message.
	ctx.StatusCode(iris.StatusOK)
	ctx.JSON(iris.Map{"message": "Customer deleted successfully"})
}
```

The `pg.DB` function returns the database instance associated with the Iris context; use it for any operation the [pg](https://github.com/kataras/pg) package API offers. The `pg.Repository` function returns a generic repository interface for the given model type, with type-safe methods for common CRUD operations. This example uses the repository interface to insert, select, update and delete customers.

## How to run the example?

1. Clone the [PG middleware repository](https://github.com/iris-contrib/middleware/tree/main/pg) and navigate to the `_examples/basic` folder.
2. Install the dependencies using `go mod tidy`.
3. Start a PostgreSQL server and create a database named `test_db`.
4. Run the main.go file using `go run main.go`.
5. Use a tool like [Postman](https://www.postman.com/) or [curl](https://curl.se/) to test the API endpoints:

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

## Conclusion

We connected Iris and PostgreSQL through PG middleware and built a small REST API for managing customers, with transactions, schema creation and validation, query tracing and error handling configured in one place.

Article published at:

1. https://dev.to/kataras/how-to-use-iris-and-postgresql-for-web-development-3kka
	- https://twitter.com/TheGoDev/status/1676701488544874498
	- https://twitter.com/TheDatabaseDev/status/1676748294595174401
2. https://medium.com/@kataras/how-to-use-iris-and-postgresql-for-web-development-e8c46d72f1e6
3. https://www.linkedin.com/pulse/how-use-iris-postgresql-web-development-gerasimos-maropoulos

If you have any feedback or questions, please open an issue or a pull request on [GitHub](https://github.com/iris-contrib/middleware/tree/main/pg).
