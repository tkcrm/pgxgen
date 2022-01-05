package unstable

// DO NOT USE. Use pgxgen.IsNotFoundError instead of directly referencing this type.
type NotFoundError struct {
	Msg string
}

func (e *NotFoundError) Error() string {
	return e.Msg
}
