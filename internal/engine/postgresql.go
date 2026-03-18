package engine

import "fmt"

type postgresql struct{}

func (e *postgresql) Name() string                      { return "postgresql" }
func (e *postgresql) ParamPlaceholder(index int) string { return fmt.Sprintf("$%d", index) }
func (e *postgresql) BoolCast(expr string) string       { return expr + "::boolean" }
func (e *postgresql) TimestampFunc() string             { return "now()" }
func (e *postgresql) SupportsReturning() bool           { return true }
func (e *postgresql) BatchInsertStrategy() string       { return "copyfrom" }
