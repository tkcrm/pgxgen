package engine

type sqlite struct{}

func (e *sqlite) Name() string                  { return "sqlite" }
func (e *sqlite) ParamPlaceholder(_ int) string { return "?" }
func (e *sqlite) BoolCast(expr string) string   { return expr }
func (e *sqlite) TimestampFunc() string         { return "datetime('now')" }
func (e *sqlite) SupportsReturning() bool       { return false }
func (e *sqlite) BatchInsertStrategy() string   { return "multi-values" }
