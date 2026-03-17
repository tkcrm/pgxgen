package engine

type mysql struct{}

func (e *mysql) Name() string                    { return "mysql" }
func (e *mysql) ParamPlaceholder(_ int) string   { return "?" }
func (e *mysql) BoolCast(expr string) string     { return expr }
func (e *mysql) TimestampFunc() string            { return "NOW()" }
func (e *mysql) SupportsReturning() bool          { return false }
func (e *mysql) BatchInsertStrategy() string      { return "multi-values" }
