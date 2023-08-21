package pg

import (
	"fmt"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/x/errors"

	"github.com/kataras/pg"
	"github.com/kataras/pg/desc"
)

// EntityController is a controller for a single entity.
// It can be used to create a RESTful API for a single entity.
// It is a wrapper around the pg.Repository.
// It can be used as a base controller for a custom controller.
// The T is the entity type (e.g. a custom type, Customer).
//
// The controller registers the following routes:
// - POST / - creates a new entity.
// - PUT / - updates an existing entity.
// - GET /{id} - gets an entity by ID.
// - DELETE /{id} - deletes an entity by ID.
// The {id} parameter is the entity ID. It can be a string, int, uint, uuid, etc.
type EntityController[T any] struct {
	repository *pg.Repository[T]

	tableName      string
	primaryKeyType desc.DataType

	// GetID returns the entity ID for GET/{id} and DELETE/{id} paths from the request Context.
	GetID func(ctx iris.Context) any

	// ErrorHandler defaults to the PG's error handler. It can be customized for this controller.
	// Setting this to nil will panic the application on the first error.
	ErrorHandler func(ctx iris.Context, err error) bool
}

// NewEntityController returns a new EntityController[T].
// The T is the entity type (e.g. a custom type, Customer).
//
// Read the type's documentation for more information.
func NewEntityController[T any](middleware *PG) *EntityController[T] {

	repo := pg.NewRepository[T](middleware.GetDB())
	errorHandler := middleware.opts.handleError

	td := repo.Table()
	primaryKey, ok := td.PrimaryKey()
	if !ok {
		panic(fmt.Sprintf("pg: entity %s does not have a primary key", td.Name))
	}

	return &EntityController[T]{
		repository:     repo,
		tableName:      td.Name,
		primaryKeyType: primaryKey.Type,
		ErrorHandler:   errorHandler,
	}
}

// Singleton returns true as this controller is a singleton.
func (c *EntityController[T]) Singleton() bool { return true }

// Configure registers the controller's routes.
// It is called automatically by the Iris API Builder when registered to the Iris Application.
func (c *EntityController[T]) Configure(r iris.Party) {
	idParamName := fmt.Sprintf("%s_id", c.tableName)
	idParam := fmt.Sprintf("/{%s}", idParamName)

	switch c.primaryKeyType {
	case desc.UUID:
		idParam = fmt.Sprintf("/{%s:uuid}", idParamName)
	case desc.CharacterVarying:
		idParam = fmt.Sprintf("/{%s:string}", idParamName)
	case desc.Integer, desc.SmallInt, desc.BigInt, desc.Serial, desc.SmallSerial, desc.BigSerial:
		idParam = fmt.Sprintf("/{%s:int}", idParamName)
	}

	if c.GetID == nil {
		c.GetID = func(ctx iris.Context) any {
			switch c.primaryKeyType {
			case desc.UUID, desc.CharacterVarying, desc.Text, desc.BitVarying, desc.CharacterArray:
				return ctx.Params().Get(idParamName)
			case desc.SmallInt:
				return ctx.Params().GetIntDefault(idParamName, 0)
			case desc.Integer, desc.BigInt, desc.Serial, desc.SmallSerial, desc.BigSerial:
				return ctx.Params().GetInt64Default(idParamName, 0)
			default:
				return ctx.Params().GetEntry(idParamName).Value()
			}
		}
	}

	r.Post("/", c.create)
	r.Put("/", c.update)
	r.Get(idParam, c.get)
	r.Delete(idParam, c.delete)
}

// handleError handles the error. It returns true if the error was handled.
func (c *EntityController[T]) handleError(ctx iris.Context, err error) bool {
	if err == nil {
		return false
	}

	return c.ErrorHandler(ctx, err)
}

// readPayload reads the request body and returns the entity.
func (c *EntityController[T]) readPayload(ctx iris.Context) (T, bool) {
	var payload T
	err := ctx.ReadJSON(&payload)
	if err != nil {
		errors.InvalidArgument.Details(ctx, "unable to parse body", err.Error())
		return payload, false
	}

	return payload, true
}

type idPayload struct {
	ID any `json:"id"`
}

func toUUIDv4(v [16]uint8) string {
	slice := v[:]
	// Modify the 7th element to have the form 4xxx
	slice[6] = (slice[6] & 0x0f) | 0x40
	// Modify the 9th element to have the form yxxx
	slice[8] = (slice[8] & 0x3f) | 0x80
	// Convert to UUIDv4 string
	s := fmt.Sprintf("%x-%x-%x-%x-%x", slice[0:4], slice[4:6], slice[6:8], slice[8:10], slice[10:])
	return s
}

// create creates a new entity.
func (c *EntityController[T]) create(ctx iris.Context) {
	entry, ok := c.readPayload(ctx)
	if !ok {
		return
	}

	var id any
	err := c.repository.InsertSingle(ctx, entry, &id)
	if c.handleError(ctx, err) {
		return
	}

	switch c.primaryKeyType {
	case desc.UUID:
		// A special case to convert from [16]uint8 to string (uuidv4). We do this in order to not accept a 2nd generic parameter of V.
		id = toUUIDv4(id.([16]uint8))
	}

	ctx.StatusCode(iris.StatusCreated)
	ctx.JSON(idPayload{ID: id})
}

// get gets an entity by ID.
func (c *EntityController[T]) get(ctx iris.Context) {
	id := c.GetID(ctx)

	entry, err := c.repository.SelectByID(ctx, id)
	if c.handleError(ctx, err) {
		return
	}

	ctx.JSON(entry)
}

// update updates an entity.
func (c *EntityController[T]) update(ctx iris.Context) {
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
func (c *EntityController[T]) delete(ctx iris.Context) {
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
