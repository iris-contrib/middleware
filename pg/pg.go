package pg

import (
	stdContext "context"
	"errors"
	"fmt"
	"time"

	"github.com/kataras/golog"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
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
	Transactional bool `yaml:"Transactional"` // If true then all requests will be executed in transaction.
	CreateSchema  bool `yaml:"CreateSchema"`  // If true then schema will be created if not exists.
	CheckSchema   bool `yaml:"CheckSchema"`   // If true then check the schema for missing tables and columns.
	//
	ErrorHandler func(ctx iris.Context, err error) // The error handler for the middleware.
}

func (o *Options) getConnString() string {
	// Format the connection string using the parameters.
	connString := fmt.Sprintf("host=%s port=%d user=%s password=%s search_path=%s dbname=%s sslmode=%s",
		o.Host, o.Port, o.User, o.Password, o.Schema, o.DBName, o.SSLMode)

	return connString
}

func (o *Options) getConnectionOptions() []pg.ConnectionOption {
	opts := make([]pg.ConnectionOption, 0)
	if o.Trace {
		logger := pgxgolog.NewLogger(golog.Default)
		opts = append(opts, pg.WithLogger(logger))
	}

	return opts
}

func (o *Options) handleError(ctx iris.Context, err error) {
	if o.ErrorHandler != nil {
		o.ErrorHandler(ctx, err)
	} else {
		ctx.StopWithError(iris.StatusInternalServerError, err)
	}
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

	iris.RegisterOnInterrupt(db.Close)

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
func NewFromDB(db *pg.DB, opts Options) *PG {
	if len(opts.getConnectionOptions()) > 0 {
		panic("pg.NewFromDB: options are not supported")
	}

	if opts.getConnString() != "" {
		panic("pg.NewFromDB: connection string is not supported")
	}

	iris.RegisterOnInterrupt(db.Close)

	return &PG{
		opts: opts,
		db:   db,
	}
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
				} else if err != nil {
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
						return
					}
				} else {
					err = tx.Commit(ctx)
					if err != nil {
						p.opts.handleError(ctx, err)
						return
					}
				}
			}()

			db = tx
		}

		ctx.Values().Set(dbContextKey, db)
		ctx.Next()
	}

	return handler
}

// Close calls the underlying *pg.DB.Close method.
func (p *PG) Close() {
	p.db.Close()
}
