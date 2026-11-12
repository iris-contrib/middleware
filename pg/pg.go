package pg

import (
	stdContext "context"
	"fmt"
	"strconv"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/rest/errors"

	// Registers the PostgreSQL error mapping of the rest/errors package:
	// no rows to 404, a duplicate key to 409, a foreign key violation and
	// invalid input to 400. DefaultErrorHandler relies on it.
	_ "github.com/kataras/iris/v14/rest/errors/pgerrors"

	"github.com/kataras/golog"

	"github.com/kataras/pg"
	pgxgolog "github.com/kataras/pgx-golog"
)

func init() {
	context.SetHandlerName("github.com/iris-contrib/middleware/pg.*", "iris-contrib.pg")
}

// Options is the configuration for the PG middleware.
// It is used to customize the connection to the database.
//
// See https://pkg.go.dev/github.com/kataras/pg for more information.
type Options struct {
	// Connection options.
	Host     string `yaml:"Host"`
	Port     int    `yaml:"Port"`
	User     string `yaml:"User"`
	Password string `yaml:"Password"`
	Schema   string `yaml:"Schema"`
	DBName   string `yaml:"DBName"`
	SSLMode  string `yaml:"SSLMode"`
	//
	Trace bool `yaml:"Trace"` // If true then database tracer with Logger will be registered.
	//
	// If true then each request runs in a transaction. It is committed when
	// the handlers return, and rolled back when one of them panics or stores
	// an error on the Context (ctx.StopWithError, ctx.SetErr). Store
	// pg.ErrIntentionalRollback to roll back a request that did not fail.
	Transactional bool `yaml:"Transactional"`
	CreateSchema  bool `yaml:"CreateSchema"` // If true then schema will be created if not exists.
	CheckSchema   bool `yaml:"CheckSchema"`  // If true then check the schema for missing tables and columns.
	//
	// The error handler for the middleware and for the EntityController.
	// The implementation can ignore the error and return false to continue to the default error handler.
	ErrorHandler func(ctx iris.Context, err error) bool
}

func (o *Options) getConnString() string {
	// Format the connection string using the parameters.
	connString := fmt.Sprintf("host=%s port=%d user=%s password=%s search_path=%s dbname=%s sslmode=%s",
		o.Host, o.Port, o.User, o.Password, o.Schema, o.DBName, o.SSLMode)

	return connString
}

// hasConnectionFields reports whether any of the connection fields is set.
func (o *Options) hasConnectionFields() bool {
	return o.Host != "" || o.Port != 0 || o.User != "" || o.Password != "" ||
		o.Schema != "" || o.DBName != "" || o.SSLMode != ""
}

func (o *Options) getConnectionOptions() []pg.ConnectionOption {
	opts := make([]pg.ConnectionOption, 0)
	if o.Trace {
		logger := pgxgolog.NewLogger(golog.Default)
		opts = append(opts, pg.WithLogger(logger))
	}

	return opts
}

// DefaultErrorHandler is the default error handler for the PG middleware.
// It answers through the framework's rest/errors package, like a handler
// that calls errors.HandleError does:
//
//   - pg.ErrNoRows: 404
//   - a duplicate key: 409
//   - a foreign key violation: 400, naming the constraint
//   - a value PostgreSQL could not parse (invalid input syntax): 400
//   - a value that is not a number (strconv.ErrSyntax): 400
//   - anything else: 500 with a generic message
//
// The database's own message, which names tables, columns and the offending
// value, is logged and not sent to the client. A nil error reports false.
var DefaultErrorHandler = func(ctx iris.Context, err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, strconv.ErrSyntax) {
		errors.InvalidArgument.Err(ctx, err)
		return true
	}

	errors.HandleError(ctx, err)
	return true
}

func (o *Options) handleError(ctx iris.Context, err error) bool {
	if err == nil {
		return false
	}

	// Keep the error on the Context, unless a handler stored one already:
	// that is what rolls the request's transaction back, and what a logger
	// registered outside the middleware reads. It is private, so nothing
	// renders it to the client.
	if ctx.GetErr() == nil {
		ctx.SetErr(iris.PrivateError(err))
	}

	if o.ErrorHandler != nil {
		if o.ErrorHandler(ctx, err) {
			return true
		}
	}

	// If error handler is not registered or returned false, call the default one.
	return DefaultErrorHandler(ctx, err)
}

// PG is the PG middleware.
// It holds the *pg.DB instance and the options.
//
// Its `Handler` method should be registered to the Iris Application.
type PG struct {
	opts Options
	db   *pg.DB
}

// New returns a new PG middleware instance.
func New(schema *pg.Schema, opts Options) *PG {
	ctx, cancel := stdContext.WithTimeout(stdContext.Background(), 10*time.Second)
	defer cancel()

	db, err := pg.Open(ctx, schema, opts.getConnString(), opts.getConnectionOptions()...)
	if err != nil {
		panic(err)
	}

	// iris.RegisterOnInterrupt(db.Close)

	if opts.CreateSchema {
		if err = db.CreateSchema(ctx); err != nil {
			panic(err)
		}
	}

	if opts.CheckSchema {
		if err = db.CheckSchema(ctx); err != nil {
			panic(err)
		}
	}

	return &PG{
		opts: opts,
		db:   db,
	}
}

// NewFromDB returns a new PG middleware instance from an existing *pg.DB.
//
// The database is already open, so the connection fields of the options
// (Host, Port, User, Password, Schema, DBName, SSLMode) and Trace must be
// left unset: it panics otherwise. Set the tracer on the *pg.DB instead.
// Transactional, CreateSchema, CheckSchema and ErrorHandler apply as in New.
func NewFromDB(db *pg.DB, opts Options) *PG {
	if opts.Trace {
		panic("pg.NewFromDB: the Trace option is not supported, register the tracer when you open the database")
	}

	if opts.hasConnectionFields() {
		panic("pg.NewFromDB: the connection fields are not supported, the database is already open")
	}

	ctx, cancel := stdContext.WithTimeout(stdContext.Background(), 10*time.Second)
	defer cancel()

	if opts.CreateSchema {
		if err := db.CreateSchema(ctx); err != nil {
			panic(err)
		}
	}

	if opts.CheckSchema {
		if err := db.CheckSchema(ctx); err != nil {
			panic(err)
		}
	}

	return &PG{
		opts: opts,
		db:   db,
	}
}

// GetDB returns the underlying *pg.DB instance.
func (p *PG) GetDB() *pg.DB {
	return p.db
}

// Close calls the underlying *pg.DB.Close method.
func (p *PG) Close() {
	p.db.Close()
}

const dbContextKey = "iris.contrib.pgdb"

// DB returns the *pg.DB binded to the "iris.contrib.pgdb" context key.
func DB(ctx iris.Context) *pg.DB {
	if v := ctx.Values().Get(dbContextKey); v != nil {
		if db, ok := v.(*pg.DB); ok {
			return db
		}
	}

	return nil
}

// Repository returns a new Repository of T type by the database instance
// binded to the request Context.
func Repository[T any](ctx iris.Context) *pg.Repository[T] {
	db := DB(ctx)
	if db == nil {
		return nil
	}

	repo := pg.NewRepository[T](db)
	return repo
}

// Handler returns a middleware which adds a *pg.DB binded to the request Context.
func (p *PG) Handler() iris.Handler {
	handler := func(ctx iris.Context) {
		db := DB(ctx) // try to get it from a previous handler (if any, it shouldn't be any but just in case)
		if db == nil {
			db = p.db
		}

		if p.opts.Transactional && !db.IsTransaction() {
			tx, err := db.Begin(ctx)
			if err != nil {
				p.opts.handleError(ctx, err)
				return
			}

			defer func() {
				if rec := recover(); rec != nil {
					_ = tx.Rollback(ctx)
					panic(rec) // re-throw panic after RollbackDatabase.
				}

				// The handlers have no return value: one that failed says so
				// by storing an error on the Context. It plays the part of
				// the error a pg.DB.InTransaction function returns.
				if err := ctx.GetErr(); err != nil {
					if errors.Is(err, pg.ErrIntentionalRollback) {
						err = tx.Rollback(ctx)
						if err != nil {
							p.opts.handleError(ctx, err)
						}
						return
					}

					rollbackErr := tx.Rollback(ctx)
					if rollbackErr != nil {
						err = fmt.Errorf("%w: %s", err, rollbackErr.Error())
						p.opts.handleError(ctx, err)
					}
					return
				}

				if err := tx.Commit(ctx); err != nil {
					p.opts.handleError(ctx, err)
				}
			}()

			db = tx
		}

		ctx.Values().Set(dbContextKey, db)
		ctx.Next()
	}

	return handler
}
