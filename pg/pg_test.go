package pg_test

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	irispg "github.com/iris-contrib/middleware/pg"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/recovery"
	"github.com/kataras/pg"
)

// Customer is the table model of the tests.
type Customer struct {
	ID   string `json:"id" pg:"type=uuid,primary"`
	Name string `json:"name" pg:"type=varchar(255)"`
}

func newSchema() *pg.Schema {
	schema := pg.NewSchema()
	schema.MustRegister("customers", Customer{})
	return schema
}

// newMiddleware returns the middleware over a database that is a fake:
// see fakePostgres.
func newMiddleware(t *testing.T, opts irispg.Options) (*irispg.PG, *fakePostgres) {
	t.Helper()

	db, fake := newFakeDB(t, newSchema())
	return irispg.NewFromDB(db, opts), fake
}

func expectStatements(t *testing.T, fake *fakePostgres, expected ...string) {
	t.Helper()

	got := fake.statements()
	if !slices.Equal(got, expected) {
		t.Fatalf("expected the database to receive\n%s\nbut it received\n%s",
			strings.Join(expected, "\n"), strings.Join(got, "\n"))
	}
}

func expectPanic(t *testing.T, fn func()) {
	t.Helper()

	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic")
		}
	}()

	fn()
}

func TestNewFromDB(t *testing.T) {
	db, _ := newFakeDB(t, newSchema())

	p := irispg.NewFromDB(db, irispg.Options{Transactional: true})
	if p.GetDB() != db {
		t.Fatal("expected GetDB to return the database the middleware was created with")
	}

	// The database is already open: options that describe how to open it are refused.
	expectPanic(t, func() { irispg.NewFromDB(db, irispg.Options{Host: "localhost"}) })
	expectPanic(t, func() { irispg.NewFromDB(db, irispg.Options{DBName: "test_db", SSLMode: "disable"}) })
	expectPanic(t, func() { irispg.NewFromDB(db, irispg.Options{Trace: true}) })
}

// New opens the database, so without a server it panics: a refused
// connection on the local machine, no network involved.
func TestNewPanicsWhenTheDatabaseIsUnreachable(t *testing.T) {
	expectPanic(t, func() {
		irispg.New(newSchema(), irispg.Options{
			Host: "127.0.0.1",
			Port: 1, // nothing listens on the tcpmux port.
			User: "postgres", Password: "x", DBName: "x", Schema: "public", SSLMode: "disable",
		})
	})
}

// The handler stores the database on the Context, where DB and Repository find it.
func TestHandler(t *testing.T) {
	p, fake := newMiddleware(t, irispg.Options{})

	app := iris.New()
	app.Get("/with", p.Handler(), func(ctx iris.Context) {
		db := irispg.DB(ctx)
		if db == nil || db != p.GetDB() || db.IsTransaction() {
			ctx.StopWithText(iris.StatusInternalServerError, "expected the plain database on the Context")
			return
		}

		if irispg.Repository[Customer](ctx) == nil {
			ctx.StopWithText(iris.StatusInternalServerError, "expected a repository")
			return
		}

		if _, err := db.Exec(ctx, "SELECT 1"); err != nil {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}

		ctx.WriteString("ok")
	})
	app.Get("/without", func(ctx iris.Context) {
		if irispg.DB(ctx) != nil || irispg.Repository[Customer](ctx) != nil {
			ctx.StopWithText(iris.StatusInternalServerError, "expected no database without the middleware")
			return
		}

		ctx.WriteString("ok")
	})

	e := httptest.New(t, app)
	e.GET("/with").Expect().Status(httptest.StatusOK).Body().IsEqual("ok")
	e.GET("/without").Expect().Status(httptest.StatusOK).Body().IsEqual("ok")

	// Not transactional: the statement runs on its own.
	expectStatements(t, fake, "SELECT 1")
}

func newTransactionalApp(p *irispg.PG) *iris.Application {
	app := iris.New()
	// The framework's recovery turns the re-thrown panic into a 500.
	app.UseRouter(recovery.Handler)

	api := app.Router("/", p.Handler())

	insert := func(ctx iris.Context) error {
		db := irispg.DB(ctx)
		if !db.IsTransaction() {
			return errors.New("expected a transaction")
		}

		_, err := db.Exec(ctx, "INSERT INTO customers (name) VALUES ('Gerasimos')")
		return err
	}

	api.Post("/ok", func(ctx iris.Context) {
		if err := insert(ctx); err != nil {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}
		ctx.StatusCode(iris.StatusCreated)
	})

	// The work is done, then the handler finds a reason to refuse the request.
	api.Post("/refused", func(ctx iris.Context) {
		if err := insert(ctx); err != nil {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}
		ctx.StopWithError(iris.StatusConflict, errors.New("the name is taken"))
	})

	api.Post("/dry-run", func(ctx iris.Context) {
		if err := insert(ctx); err != nil {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}
		ctx.SetErr(pg.ErrIntentionalRollback)
		ctx.WriteString("validated, nothing saved")
	})

	api.Post("/panic", func(ctx iris.Context) {
		_ = insert(ctx)
		panic("something went very wrong")
	})

	return app
}

const insertStatement = "INSERT INTO customers (name) VALUES ('Gerasimos')"

// expectTransaction expects the statements to have arrived in that order
// and on one connection, as the statements of a transaction do.
func expectTransaction(t *testing.T, fake *fakePostgres, expected ...string) {
	t.Helper()

	expectStatements(t, fake, expected...)
	if n := fake.connections(); n != 1 {
		t.Fatalf("expected the statements of a transaction on one connection, they arrived on %d", n)
	}
}

// A request that succeeds is committed.
func TestTransactionalCommit(t *testing.T) {
	p, fake := newMiddleware(t, irispg.Options{Transactional: true})
	e := httptest.New(t, newTransactionalApp(p))

	e.POST("/ok").Expect().Status(httptest.StatusCreated)
	expectTransaction(t, fake, "begin", insertStatement, "commit")
}

// A request whose handler stored an error on the Context is rolled back.
func TestTransactionalRollbackOnError(t *testing.T) {
	p, fake := newMiddleware(t, irispg.Options{Transactional: true})
	e := httptest.New(t, newTransactionalApp(p))

	e.POST("/refused").Expect().Status(httptest.StatusConflict).Body().IsEqual("the name is taken")
	expectTransaction(t, fake, "begin", insertStatement, "rollback")
}

// pg.ErrIntentionalRollback rolls back without turning the response into a failure.
func TestTransactionalIntentionalRollback(t *testing.T) {
	p, fake := newMiddleware(t, irispg.Options{Transactional: true})
	e := httptest.New(t, newTransactionalApp(p))

	e.POST("/dry-run").Expect().Status(httptest.StatusOK).Body().IsEqual("validated, nothing saved")
	expectTransaction(t, fake, "begin", insertStatement, "rollback")
}

// A panic rolls back and is thrown again for the recovery middleware.
func TestTransactionalRollbackOnPanic(t *testing.T) {
	p, fake := newMiddleware(t, irispg.Options{Transactional: true})
	e := httptest.New(t, newTransactionalApp(p))

	e.POST("/panic").Expect().Status(httptest.StatusInternalServerError)
	expectTransaction(t, fake, "begin", insertStatement, "rollback")
}

// A statement that fails aborts the transaction: the handler reports the
// failure and the transaction is rolled back.
func TestTransactionalRollbackOnFailedStatement(t *testing.T) {
	p, fake := newMiddleware(t, irispg.Options{Transactional: true})
	fake.setRespond(func(sql string) []pgproto3.BackendMessage {
		if strings.HasPrefix(sql, "INSERT") {
			return []pgproto3.BackendMessage{pgError("23505", "duplicate key value violates unique constraint")}
		}
		return nil
	})

	e := httptest.New(t, newTransactionalApp(p))
	e.POST("/ok").Expect().Status(httptest.StatusInternalServerError)
	expectTransaction(t, fake, "begin", insertStatement, "rollback")
}

// When the transaction cannot start the error handler answers
// and the route's handler does not run.
func TestTransactionalBeginError(t *testing.T) {
	var handled error

	p, fake := newMiddleware(t, irispg.Options{
		Transactional: true,
		ErrorHandler: func(ctx iris.Context, err error) bool {
			handled = err
			ctx.StopWithText(iris.StatusServiceUnavailable, "database unavailable")
			return true
		},
	})
	fake.setRespond(func(sql string) []pgproto3.BackendMessage {
		if sql == "begin" {
			return []pgproto3.BackendMessage{pgError("53300", "too many connections")}
		}
		return nil
	})

	e := httptest.New(t, newTransactionalApp(p))
	e.POST("/ok").Expect().Status(httptest.StatusServiceUnavailable).Body().IsEqual("database unavailable")

	if handled == nil || !strings.Contains(handled.Error(), "too many connections") {
		t.Fatalf("expected the begin error to reach the error handler, got %v", handled)
	}
	expectStatements(t, fake, "begin")
}

// When the commit fails the error handler is given the error.
func TestTransactionalCommitError(t *testing.T) {
	var handled error

	p, fake := newMiddleware(t, irispg.Options{
		Transactional: true,
		ErrorHandler: func(ctx iris.Context, err error) bool {
			handled = err
			return true
		},
	})
	fake.setRespond(func(sql string) []pgproto3.BackendMessage {
		if sql == "commit" {
			return []pgproto3.BackendMessage{pgError("40001", "could not serialize access")}
		}
		return nil
	})

	e := httptest.New(t, newTransactionalApp(p))
	e.POST("/ok").Expect()

	if handled == nil || !strings.Contains(handled.Error(), "could not serialize access") {
		t.Fatalf("expected the commit error to reach the error handler, got %v", handled)
	}
}

// The default error handler answers with the framework's REST error document
// and keeps what the database said about the schema and the data to itself.
func TestDefaultErrorHandler(t *testing.T) {
	const secret = "customers_email_key (email)=(makis@example.com)"

	tests := []struct {
		name   string
		err    error
		status int
	}{
		{"no rows", fmt.Errorf("select: %w", pg.ErrNoRows), httptest.StatusNotFound},
		{"duplicate key", &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint " + secret}, httptest.StatusConflict},
		{"foreign key", &pgconn.PgError{Code: "23503", Message: "violates foreign key constraint " + secret, ConstraintName: "orders_customer_fk"}, httptest.StatusBadRequest},
		{"invalid input", &pgconn.PgError{Code: "22P02", Message: `invalid input syntax for type uuid: "x"`}, httptest.StatusBadRequest},
		{"not a number", fmt.Errorf("id: %w", strconv.ErrSyntax), httptest.StatusBadRequest},
		{"anything else", errors.New("connection reset by peer while reading " + secret), httptest.StatusInternalServerError},
	}

	app := iris.New()
	app.Get("/{n:int}", func(ctx iris.Context) {
		if !irispg.DefaultErrorHandler(ctx, tests[ctx.Params().GetInt("n")].err) {
			ctx.WriteString("not handled")
		}
	})

	e := httptest.New(t, app)
	for i, tt := range tests {
		r := e.GET("/" + strconv.Itoa(i)).Expect().Status(tt.status)
		r.JSON().Object().Value("http_error_code").Object().HasValue("status", tt.status)

		if body := r.Body().Raw(); strings.Contains(body, "makis@example.com") || strings.Contains(body, "customers_email_key") {
			t.Errorf("%s: the response repeats what the database said: %s", tt.name, body)
		}
	}
}

// Options.ErrorHandler runs first. When it reports false the default one answers.
func TestErrorHandlerOption(t *testing.T) {
	errQuota := errors.New("quota exceeded")

	p, fake := newMiddleware(t, irispg.Options{
		Transactional: true,
		ErrorHandler: func(ctx iris.Context, err error) bool {
			if strings.Contains(err.Error(), "quota exceeded") {
				ctx.StopWithText(iris.StatusPaymentRequired, errQuota.Error())
				return true
			}
			return false
		},
	})

	var answer []pgproto3.BackendMessage
	fake.setRespond(func(sql string) []pgproto3.BackendMessage {
		if sql == "begin" {
			return answer
		}
		return nil
	})

	e := httptest.New(t, newTransactionalApp(p))

	answer = []pgproto3.BackendMessage{pgError("P0001", "quota exceeded")}
	e.POST("/ok").Expect().Status(iris.StatusPaymentRequired).Body().IsEqual("quota exceeded")

	answer = []pgproto3.BackendMessage{pgError("53300", "too many connections")}
	e.POST("/ok").Expect().Status(httptest.StatusInternalServerError).
		JSON().Object().Value("http_error_code").Object().HasValue("status", httptest.StatusInternalServerError)
}

func TestIsErrNoRows(t *testing.T) {
	if !irispg.IsErrNoRows(fmt.Errorf("wrapped: %w", irispg.ErrNoRows)) {
		t.Fatal("expected a wrapped ErrNoRows to be reported")
	}

	if irispg.IsErrNoRows(nil) || irispg.IsErrNoRows(errors.New("other")) {
		t.Fatal("expected nil and other errors not to be reported")
	}
}
