package pg

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/x/errors"

	"github.com/kataras/pg"
)

// EntityController is a controller for a single entity.
// It can be used to create a RESTful API for a single entity.
// It is a wrapper around the pg.Repository.
// It can be used as a base controller for a custom controller.
// The T is the entity type (e.g. a custom type, Customer) and V is the ID type (e.g. string).
//
// The controller registers the following routes:
// - POST / - creates a new entity.
// - PUT / - updates an existing entity.
// - GET /{id} - gets an entity by ID.
// - DELETE /{id} - deletes an entity by ID.
// The {id} parameter is the entity ID. It can be a string, int, uint, uuid, etc.
type EntityController[T any, V comparable] struct {
	repository *pg.Repository[T]

	// GetID returns the entity ID for GET/{id} and DELETE/{id} paths from the request Context.
	GetID func(ctx iris.Context) V

	// ErrorHandler defaults to the PG's error handler. It can be customized for this controller.
	// Setting this to nil will panic the application on the first error.
	ErrorHandler func(ctx iris.Context, err error) bool
}

// NewEntityController returns a new EntityController[T, V].
// The T is the entity type (e.g. a custom type, Customer) and V is the ID type (e.g. string).
//
// Read the type's documentation for more information.
func NewEntityController[T any, V comparable](middleware *PG) *EntityController[T, V] {
	repo := pg.NewRepository[T](middleware.GetDB())
	errorHandler := middleware.opts.handleError

	return &EntityController[T, V]{
		repository:   repo,
		ErrorHandler: errorHandler,
	}
}

// Configure registers the controller's routes.
// It is called automatically by the Iris API Builder when registered to the Iris Application.
func (c *EntityController[T, V]) Configure(r iris.Party) {
	var v V
	typ := reflect.TypeOf(v)
	idParamName := fmt.Sprintf("%s_id", strings.ToLower(typ.Name()))
	idParam := fmt.Sprintf("/{%s}", idParamName)
	switch typ.Kind() {
	case reflect.String:
		idParam = fmt.Sprintf("/{%s:uuid}", idParamName)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		idParam = fmt.Sprintf("/{%s:int}", idParamName)
	}

	if c.GetID == nil {
		c.GetID = func(ctx iris.Context) V {
			return ctx.Params().GetEntry(idParamName).Value().(V)
		}
	}

	r.Post("/", c.create)
	r.Put("/", c.update)
	r.Get(idParam, c.get)
	r.Delete(idParam, c.delete)
}

// handleError handles the error. It returns true if the error was handled.
func (c *EntityController[T, V]) handleError(ctx iris.Context, err error) bool {
	if err == nil {
		return false
	}

	return c.ErrorHandler(ctx, err)
}

// readPayload reads the request body and returns the entity.
func (c *EntityController[T, V]) readPayload(ctx iris.Context) (T, bool) {
	var payload T
	err := ctx.ReadJSON(&payload)
	if err != nil {
		errors.InvalidArgument.Details(ctx, "unable to parse body", err.Error())
		return payload, false
	}

	return payload, true
}

type idPayload[T comparable] struct {
	ID T `json:"id"`
}

// create creates a new entity.
func (c *EntityController[T, V]) create(ctx iris.Context) {
	entry, ok := c.readPayload(ctx)
	if !ok {
		return
	}

	var id V
	err := c.repository.InsertSingle(ctx, entry, &id)
	if c.handleError(ctx, err) {
		return
	}

	ctx.StatusCode(iris.StatusCreated)
	ctx.JSON(idPayload[V]{ID: id})
}

// get gets an entity by ID.
func (c *EntityController[T, V]) get(ctx iris.Context) {
	id := c.GetID(ctx)

	entry, err := c.repository.SelectByID(ctx, id)
	if c.handleError(ctx, err) {
		return
	}

	ctx.JSON(entry)
}

// update updates an entity.
func (c *EntityController[T, V]) update(ctx iris.Context) {
	entry, ok := c.readPayload(ctx)
	if !ok {
		return
	}

	// patch-like.
	onlyColumns := ctx.URLParamSlice("columns")

	var (
		n   int64
		err error
	)
	if len(onlyColumns) > 0 {
		n, err = c.repository.UpdateOnlyColumns(ctx, onlyColumns, entry)
	} else {
		// put-like.
		n, err = c.repository.Update(ctx, entry)
	}
	if c.handleError(ctx, err) {
		return
	}

	if n == 0 {
		errors.NotFound.Message(ctx, "resource not found")
		return
	}

	ctx.StatusCode(iris.StatusNoContent)
	// ctx.StatusCode(iris.StatusOK)
	// ctx.JSON(iris.Map{"message": "resource updated successfully"})
}

// delete deletes an entity by ID.
func (c *EntityController[T, V]) delete(ctx iris.Context) {
	id := c.GetID(ctx)

	ok, err := c.repository.DeleteByID(ctx, id)
	if c.handleError(ctx, err) {
		return
	}

	if !ok {
		errors.NotFound.Details(ctx, "resource not found", err.Error())
		return
	}

	ctx.StatusCode(iris.StatusNoContent)
}
