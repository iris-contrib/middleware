# How to Use Iris and PostgreSQL for Web Development

[Iris](https://www.iris-go.com/) is a fast and lightweight web framework for Go that offers a rich set of features and a high-performance engine. PostgreSQL is a powerful and reliable relational database system that supports advanced data types and functions. Together, they can form a solid foundation for building modern web applications.

But how can you connect Iris and PostgreSQL in a simple and type-safe way? How can you perform common database operations without writing too much boilerplate code? How can you handle transactions, schema creation, query tracing and error handling in a consistent manner?

The answer is PG middleware, a package for Iris that provides easy and type-safe access to PostgreSQL database. In this article, we will show you how to use PG middleware to create a simple REST API for managing customers.

## What is PG middleware?

PG middleware is a package for Iris web framework that provides easy and type-safe access to PostgreSQL database. It has the following features:

- It supports PostgreSQL 9.5 and above.
- It uses [pg](https://github.com/kataras/pg) package and [pgx](https://github.com/jackc/pgx) driver under the hood.
- It supports transactions, schema creation and validation, query tracing and error handling.
- It allows registering custom types and table models using a schema object.
- It provides a generic repository interface for common CRUD operations.

## How to install PG middleware?

To install PG middleware, you need to use the following command:

```sh
go get github.com/iris-contrib/middleware/pg@master
```

## How to use PG middleware?

To use PG middleware, you need to follow these steps:

1. Import the package in your code:

```go
import (
	"github.com/kataras/iris/v12"

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

The `json` tag defines how the struct fields are encoded or decoded as JSON. The `pg` tag defines how the struct fields are mapped to the database table columns. You can specify the column type, constraints, indexes and other options using the `pg` tag. Read more at [pg](https://github.com/kataras/pg) repository.

3. Create a schema object and register your models:

```go
schema := pg.NewSchema()
schema.MustRegister("customers", Customer{})
```

The schema object is used to store the metadata of your database tables and models. You need to register your models with the schema object using the `MustRegister` method. The first argument is the table name, and the second argument is the model type.

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
	ErrorHandler: func(ctx iris.Context, err error) {
		ctx.StopWithError(iris.StatusInternalServerError, err)
	},
}

p := pg.New(schema, opts)
```

The options struct defines the configuration parameters for connecting to the database and using the middleware features. You need to specify the host, port, user, password, dbname, schema and sslmode fields for establishing the connection. You can also enable or disable the transactional feature, which wraps each request handler in a database transaction. You can also enable or disable the trace feature, which logs each query executed by the middleware. You can also enable or disable the createSchema feature, which creates the schema if it doesn't exist in the database. You can also enable or disable the checkSchema feature, which checks the schema for missing tables and columns and reports any discrepancies. You can also provide an errorHandler function, which handles any errors occurred during the middleware execution.

The `New` function creates a new PG middleware instance with the given schema and options.

5. Attach the middleware handler to your Iris app or routes:

```go
app := iris.New()

postgresMiddleware := newPostgresMiddleware()

{
	customerAPI := app.Party("/api/customer", postgresMiddleware)
	customerAPI.Post("/", createCustomer)
	customerAPI.Get("/{id:uuid}", getCustomer)
	customerAPI.Put("/{id:uuid}", updateCustomer)
	customerAPI.Delete("/{id:uuid}", deleteCustomer)
}
```

The middleware handler is a function that takes an Iris context and calls the next handler in the chain. You can attach the middleware handler to your Iris app or routes using the `Use` or `Party` methods. In this example, we create a subrouter for the customer API and apply the middleware handler to it.

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

func updateCustomer(ctx iris.Context) {
    // Get the id from the path parameter.
	id := ctx.Params().Get("id")

    var payload = struct {
        Name string `json:"name"`
    }{}
    err := ctx.ReadJSON(&payload)
    if err != nil {
        ctx.StopWithError(iris.StatusBadRequest, err)
        return
    }

    // Get the repository of Customer type through pg.Repository middleware package-level function.
    customers := pg.Repository[Customer](ctx)

    // Update the customer by the id and name.
    customer := Customer{
        ID: id,
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
	id := ctx.Params().Get("id")

    // Get the repository of Customer type through pg.Repository middleware package-level function.
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

The `pg.DB` function returns the current database instance associated with the Iris context. You can use this instance to perform any database operations using the `pg` package API. The `pg.Repository` function returns a generic repository interface for the given model type associated with the Iris context. You can use this interface to perform common CRUD operations using type-safe methods. In this example, we use the repository interface to insert and select customers.

## How to run the example?

To run the example, you need to:

1. Clone the [PG middleware repository](https://github.com/iris-contrib/middleware/tree/master/pg) and navigate to the `_examples/basic` folder.
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

In this article, we have shown you how to use PG middleware to connect Iris and PostgreSQL in a simple and type-safe way. We have also demonstrated how to use PG middleware features such as transactions, schema creation and validation, query tracing and error handling. We have also shown how to use PG middleware to create a simple REST API for managing customers.

Article published at:

1. https://dev.to/kataras/how-to-use-iris-and-postgresql-for-web-development-3kka
	- https://twitter.com/TheGoDev/status/1676701488544874498
	- https://twitter.com/TheDatabaseDev/status/1676748294595174401
2. https://medium.com/@kataras/how-to-use-iris-and-postgresql-for-web-development-e8c46d72f1e6
3. https://www.linkedin.com/pulse/how-use-iris-postgresql-web-development-gerasimos-maropoulos

We hope you find PG middleware useful and easy to use. If you have any feedback or questions, please feel free to open an issue or a pull request on [GitHub](https://github.com/iris-contrib/middleware/tree/master/pg). Thank you for reading!🙏
