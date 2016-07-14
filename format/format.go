// Package format implements file format-specific operations for plugs. The
// plug (and associated) types here assist in reading/writing proprietary file
// formats from manufacturers.
package format

import (
	"fmt"
	"reflect"
)

// TODO: clean up error types

type Decoder interface {
	Decode(interface{}) error
	IsBlank() bool
}

type DecodeError struct {
	msg string
}

func (e DecodeError) Error() string {
	return e.msg
}

type DecodeDestError struct {
	t reflect.Type
}

func (e DecodeDestError) Error() string {
	return fmt.Sprintf("Invalid decode destination type %s", e.t)
}

func isNullByte(r rune) bool {
	return r == 0
}
