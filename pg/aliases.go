package pg

import (
	"errors"

	"github.com/kataras/pg"
)

// NewSchema returns a new Schema instance, it's a shortcut of pg.NewSchema.
var NewSchema = pg.NewSchema

// ErrNoRows is a type alias of pg.ErrNoRows.
var ErrNoRows = pg.ErrNoRows

// IsErrNoRows reports whether the error is of type pg.ErrNoRows.
func IsErrNoRows(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, ErrNoRows)
}
