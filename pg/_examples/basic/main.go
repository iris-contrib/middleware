package main

import (
	"github.com/kataras/iris/v12"

	"github.com/iris-contrib/middleware/pg"
)

// The Customer database table model.
type Customer struct {
	ID   string `json:"id" pg:"type=uuid,primary"`
	Name string `json:"name" pg:"type=varchar(255)"`
}

func newPostgresMiddleware() iris.Handler {
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
		ErrorHandler: func(ctx iris.Context, err error) {
			ctx.StopWithError(iris.StatusInternalServerError, err)
		},
	}

	p := pg.New(schema, opts)
	// OR pg.NewFromDB(db, pg.Options{Transactional: true})
	return p.Handler()
}

func main() {
	app := iris.New()

	postgresMiddleware := newPostgresMiddleware()

	customerAPI := app.Party("/api/customer", postgresMiddleware)
	customerAPI.Post("/", createCustomer)
	customerAPI.Get("/{id:uuid}", getCustomer)
	customerAPI.Put("/{id:uuid}", updateCustomer)
	customerAPI.Delete("/{id:uuid}", deleteCustomer)

	customerAPI.PartyConfigure("/")
	/*
		Create Customer:

		curl --location 'http://localhost:8080/api/customer' \
		--header 'Content-Type: application/json' \
		--data '{"name": "Gerasimos"}'

		Response:
		{
			"id": "a2657a3c-e5f7-43f8-adae-01bca01b3325"
		}

		Get Customer by ID:

		curl --location 'http://localhost:8080/api/customer/a2657a3c-e5f7-43f8-adae-01bca01b3325'

		Response:
		{
			"id": "a2657a3c-e5f7-43f8-adae-01bca01b3325",
			"name": "Gerasimos"
		}

	*/
	app.Listen(":8080")
}

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
		ID:   id,
		Name: payload.Name,
	}

	_, err = customers.UpdateOnlyColumns(ctx, []string{"name"}, customer)
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
