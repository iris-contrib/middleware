package pg_test

import (
	"errors"
	"strings"
	"testing"

	irispg "github.com/iris-contrib/middleware/pg"

	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

const customerID = "a2657a3c-e5f7-43f8-adae-01bca01b3325"

// Product has a numeric primary key and validates itself.
type Product struct {
	ID   int64  `json:"id" pg:"type=bigint,primary"`
	Name string `json:"name" pg:"type=varchar(255)"`
}

func (p Product) Validate() error {
	if p.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

// database answers the statements of the entity controller like a table
// that holds one customer would.
func database(sql string) []pgproto3.BackendMessage {
	switch {
	case strings.HasPrefix(sql, "INSERT"):
		return rows([]column{{"id", oidUUID}}, []string{customerID})
	case strings.HasPrefix(sql, "SELECT") && strings.Contains(sql, customerID):
		return rows([]column{{"id", oidUUID}, {"name", oidText}}, []string{customerID, "Gerasimos"})
	case strings.HasPrefix(sql, "SELECT"):
		return rows([]column{{"id", oidUUID}, {"name", oidText}})
	case strings.HasPrefix(sql, "UPDATE") && strings.Contains(sql, customerID):
		return commandTag("UPDATE 1")
	case strings.HasPrefix(sql, "UPDATE"):
		return commandTag("UPDATE 0")
	case strings.HasPrefix(sql, "DELETE") && strings.Contains(sql, customerID):
		return commandTag("DELETE 1")
	case strings.HasPrefix(sql, "DELETE"):
		return commandTag("DELETE 0")
	}

	return nil
}

// newControllerApp registers the customer controller under /api/customer,
// as the controller example does.
func newControllerApp(t *testing.T, opts irispg.Options, configure func(*irispg.EntityController[Customer])) (*httptest.Expect, *fakePostgres) {
	t.Helper()

	p, fake := newMiddleware(t, opts)
	fake.setRespond(database)

	controller := irispg.NewEntityController[Customer](p)
	if configure != nil {
		configure(controller)
	}

	app := iris.New()
	app.Router("/api/customer", controller)

	return httptest.New(t, app), fake
}

func TestEntityControllerSchema(t *testing.T) {
	e, fake := newControllerApp(t, irispg.Options{}, nil)

	fields := e.GET("/api/customer/schema").Expect().Status(httptest.StatusOK).
		JSON().Object().Value("fields").Array()
	fields.Length().IsEqual(2)
	fields.Value(0).Object().HasValue("name", "id").HasValue("data_type", "uuid").HasValue("required", true)
	fields.Value(1).Object().HasValue("name", "name").HasValue("type", "string")

	// The schema comes from the table definition, not from the database.
	expectStatements(t, fake)

	e, _ = newControllerApp(t, irispg.Options{}, func(c *irispg.EntityController[Customer]) {
		c.WithoutSchemaRoute()
	})
	e.GET("/api/customer/schema").Expect().Status(httptest.StatusNotFound)
}

func TestEntityControllerCreate(t *testing.T) {
	e, fake := newControllerApp(t, irispg.Options{}, nil)

	e.POST("/api/customer").WithJSON(Customer{Name: "Gerasimos"}).
		Expect().Status(httptest.StatusCreated).
		JSON().Object().HasValue("id", customerID)
	expectStatements(t, fake, `INSERT INTO "public"."customers" (name) VALUES( 'Gerasimos' ) RETURNING id;`)

	// A body that is not JSON is refused before the database is asked anything.
	fake.reset()
	r := e.POST("/api/customer").WithHeader("Content-Type", "application/json").WithText("{not json").
		Expect().Status(httptest.StatusBadRequest)
	r.JSON().Object().Value("http_error_code").Object().HasValue("canonical_name", "INVALID_ARGUMENT")
	expectStatements(t, fake)
}

func TestEntityControllerAfterPayloadRead(t *testing.T) {
	e, fake := newControllerApp(t, irispg.Options{}, func(c *irispg.EntityController[Customer]) {
		c.AfterPayloadRead = func(ctx iris.Context, payload Customer) (Customer, bool) {
			if payload.Name == "" {
				ctx.StopWithText(iris.StatusUnprocessableEntity, "a customer needs a name")
				return payload, false
			}

			payload.Name = strings.ToUpper(payload.Name)
			return payload, true
		}
	})

	e.POST("/api/customer").WithJSON(Customer{}).
		Expect().Status(httptest.StatusUnprocessableEntity).Body().IsEqual("a customer needs a name")
	expectStatements(t, fake)

	e.POST("/api/customer").WithJSON(Customer{Name: "Gerasimos"}).
		Expect().Status(httptest.StatusCreated)
	expectStatements(t, fake, `INSERT INTO "public"."customers" (name) VALUES( 'GERASIMOS' ) RETURNING id;`)
}

func TestEntityControllerGet(t *testing.T) {
	e, _ := newControllerApp(t, irispg.Options{}, nil)

	e.GET("/api/customer/"+customerID).Expect().Status(httptest.StatusOK).
		JSON().Object().HasValue("id", customerID).HasValue("name", "Gerasimos")

	// No such row: the default error handler answers 404.
	e.GET("/api/customer/00000000-0000-4000-8000-000000000000").Expect().Status(httptest.StatusNotFound).
		JSON().Object().Value("http_error_code").Object().HasValue("canonical_name", "NOT_FOUND")

	// The id is a uuid path parameter: anything else matches no route.
	e.GET("/api/customer/not-a-uuid").Expect().Status(httptest.StatusNotFound).Body().IsEmpty()
}

func TestEntityControllerUpdate(t *testing.T) {
	e, fake := newControllerApp(t, irispg.Options{}, nil)

	e.PUT("/api/customer").WithJSON(Customer{ID: customerID, Name: "Makis"}).
		Expect().Status(httptest.StatusNoContent)
	e.PUT("/api/customer").WithQuery("columns", "name").WithJSON(Customer{ID: customerID, Name: "Makis"}).
		Expect().Status(httptest.StatusNoContent)

	statements := fake.statements()
	if len(statements) != 2 || !strings.HasPrefix(statements[0], `UPDATE "customers" SET "name" =`) {
		t.Fatalf("expected two UPDATE statements, got %v", statements)
	}

	// No row was updated.
	e.PUT("/api/customer").WithJSON(Customer{ID: "00000000-0000-4000-8000-000000000000", Name: "Nobody"}).
		Expect().Status(httptest.StatusNotFound).
		JSON().Object().HasValue("message", "resource not found")
}

func TestEntityControllerDelete(t *testing.T) {
	e, _ := newControllerApp(t, irispg.Options{}, nil)

	e.DELETE("/api/customer/" + customerID).Expect().Status(httptest.StatusNoContent)

	// No row was deleted.
	e.DELETE("/api/customer/00000000-0000-4000-8000-000000000000").Expect().Status(httptest.StatusNotFound).
		JSON().Object().HasValue("message", "resource not found")
}

// A database failure goes through the error handler of the middleware,
// which answers without repeating what the database said.
func TestEntityControllerDatabaseError(t *testing.T) {
	e, fake := newControllerApp(t, irispg.Options{}, nil)
	fake.setRespond(func(sql string) []pgproto3.BackendMessage {
		return []pgproto3.BackendMessage{&pgproto3.ErrorResponse{
			Severity:       "ERROR",
			Code:           "23505",
			Message:        `duplicate key value violates unique constraint "customers_name_key"`,
			Detail:         "Key (name)=(Gerasimos) already exists.",
			ConstraintName: "customers_name_key",
		}}
	})

	r := e.POST("/api/customer").WithJSON(Customer{Name: "Gerasimos"}).
		Expect().Status(httptest.StatusConflict)
	r.JSON().Object().Value("http_error_code").Object().HasValue("canonical_name", "ALREADY_EXISTS")

	if body := r.Body().Raw(); strings.Contains(body, "customers_name_key") || strings.Contains(body, "Gerasimos") {
		t.Fatalf("the response repeats what the database said: %s", body)
	}
}

func TestEntityControllerErrorHandler(t *testing.T) {
	e, _ := newControllerApp(t, irispg.Options{}, func(c *irispg.EntityController[Customer]) {
		c.ErrorHandler = func(ctx iris.Context, err error) bool {
			if irispg.IsErrNoRows(err) {
				ctx.StopWithText(iris.StatusGone, "gone")
				return true
			}
			return irispg.DefaultErrorHandler(ctx, err)
		}
	})

	e.GET("/api/customer/00000000-0000-4000-8000-000000000000").
		Expect().Status(iris.StatusGone).Body().IsEqual("gone")
}

func TestEntityControllerGetID(t *testing.T) {
	e, fake := newControllerApp(t, irispg.Options{}, func(c *irispg.EntityController[Customer]) {
		// Whatever the path says, the customer is the one of the signed in user.
		c.GetID = func(ctx iris.Context) any {
			return customerID
		}
	})

	e.GET("/api/customer/11111111-1111-4111-8111-111111111111").Expect().Status(httptest.StatusOK).
		JSON().Object().HasValue("id", customerID)

	statements := fake.statements()
	if len(statements) != 1 || !strings.Contains(statements[0], customerID) {
		t.Fatalf("expected the statement to select by the id of GetID, got %v", statements)
	}
}

// An entity with a numeric primary key gets a numeric path parameter, and
// one that implements Validate is validated when its body is read.
func TestEntityControllerNumericKeyAndValidation(t *testing.T) {
	schema := newSchema()
	schema.MustRegister("products", Product{})

	db, fake := newFakeDB(t, schema)
	fake.setRespond(func(sql string) []pgproto3.BackendMessage {
		if strings.HasPrefix(sql, "DELETE") {
			return commandTag("DELETE 1")
		}
		return nil
	})

	app := iris.New()
	app.Router("/api/product", irispg.NewEntityController[Product](irispg.NewFromDB(db, irispg.Options{})))

	e := httptest.New(t, app)

	e.DELETE("/api/product/42").Expect().Status(httptest.StatusNoContent)
	statements := fake.statements()
	if len(statements) != 1 || !strings.Contains(statements[0], "42") {
		t.Fatalf("expected the id 42 in the DELETE statement, got %v", statements)
	}

	// Not a number: no route matches.
	e.DELETE("/api/product/forty-two").Expect().Status(httptest.StatusNotFound)

	fake.reset()
	e.POST("/api/product").WithJSON(Product{}).Expect().Status(httptest.StatusBadRequest).
		Body().Contains("name is required")
	expectStatements(t, fake)
}

// A table without a primary key cannot be served by id.
func TestNewEntityControllerRequiresPrimaryKey(t *testing.T) {
	type Note struct {
		Text string `json:"text" pg:"type=varchar(255)"`
	}

	schema := newSchema()
	schema.MustRegister("notes", Note{})

	db, _ := newFakeDB(t, schema)
	p := irispg.NewFromDB(db, irispg.Options{})

	expectPanic(t, func() { irispg.NewEntityController[Note](p) })
}
