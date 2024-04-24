package pg

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/x/errors"
	"github.com/kataras/iris/v12/x/jsonx"

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
// - GET /schema - returns the entity's JSON schema.
// - POST / - creates a new entity.
// - PUT / - updates an existing entity.
// - GET /{id} - gets an entity by ID.
// - DELETE /{id} - deletes an entity by ID.
// The {id} parameter is the entity ID. It can be a string, int, uint, uuid, etc.
type EntityController[T any] struct {
	iris.Singleton

	repository *pg.Repository[T]

	tableName      string
	primaryKeyType desc.DataType

	disableSchemaRoute bool

	// GetID returns the entity ID for GET/{id} and DELETE/{id} paths from the request Context.
	GetID func(ctx iris.Context) any

	// ErrorHandler defaults to the PG's error handler. It can be customized for this controller.
	// Setting this to nil will panic the application on the first error.
	ErrorHandler func(ctx iris.Context, err error) bool

	// AfterPayloadRead is called after the payload is read.
	// It can be used to validate the payload or set default fields based on the request Context.
	AfterPayloadRead func(ctx iris.Context, payload T) (T, bool)
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

	controller := &EntityController[T]{
		repository:     repo,
		tableName:      td.Name,
		primaryKeyType: primaryKey.Type,
		ErrorHandler:   errorHandler,
	}

	return controller
}

// WithoutSchemaRoute disables the GET /schema route.
func (c *EntityController[T]) WithoutSchemaRoute() *EntityController[T] {
	c.disableSchemaRoute = true
	return c
}

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

	if !c.disableSchemaRoute {
		jsonSchema := newJSONSchema[T](c.repository.Table())
		r.Get("/schema", c.getSchema(jsonSchema))
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
		errors.HandleError(ctx, err)
		return payload, false
	}

	if c.AfterPayloadRead != nil {
		return c.AfterPayloadRead(ctx, payload)
	}

	return payload, true
}

type jsonSchema[T any] struct {
	Description string                `json:"description,omitempty"`
	Types       []jsonSchemaFieldType `json:"types,omitempty"`
	Fields      []jsonSchemaField     `json:"fields"`
}

type jsonSchemaFieldType struct {
	Name    string `json:"name"`
	Example any    `json:"example,omitempty"`
}

type jsonSchemaField struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type"`
	DataType    string `json:"data_type"`
	Required    bool   `json:"required"`
}

func getJSONTag(t reflect.Type, fieldIndex []int) (string, bool) {
	if t.Kind() != reflect.Struct {
		return "", false
	}

	f := t.FieldByIndex(fieldIndex)
	jsonTag := f.Tag.Get("json")
	if jsonTag == "" {
		return "", false
	}

	return strings.Split(jsonTag, ",")[0], true
}

func newJSONSchema[T any](td *desc.Table) *jsonSchema[T] {
	var fieldTypes []jsonSchemaFieldType
	seenFieldTypes := make(map[reflect.Type]struct{})

	fields := make([]jsonSchemaField, 0, len(td.Columns))
	for _, col := range td.Columns {
		fieldName, ok := getJSONTag(col.Table.StructType, col.FieldIndex)
		if !ok {
			fieldName = col.Name
		}

		// Get the field type examples.
		if _, seen := seenFieldTypes[col.FieldType]; !seen {
			seenFieldTypes[col.FieldType] = struct{}{}

			colValue := reflect.New(col.FieldType).Interface()
			if exampler, ok := colValue.(jsonx.Exampler); ok {
				exampleValues := exampler.ListExamples()
				fieldTypes = append(fieldTypes, jsonSchemaFieldType{
					Name:    col.FieldType.String(),
					Example: exampleValues,
				})
			}
		}

		field := jsonSchemaField{
			// Here we want the json tag name, not the column name.
			Name:        fieldName,
			Description: col.Description,
			Type:        col.FieldType.String(),
			DataType:    col.Type.String(),
			Required:    !col.Nullable,
		}

		fields = append(fields, field)
	}

	return &jsonSchema[T]{
		Description: td.Description,
		Types:       fieldTypes,
		Fields:      fields,
	}
}

func (c *EntityController[T]) getSchema(s *jsonSchema[T]) iris.Handler {
	return func(ctx iris.Context) {
		ctx.JSON(s)
	}
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
