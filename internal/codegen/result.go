package codegen

// Action represents what happened to a file.
type Action string

const (
	ActionCreate    Action = "create"
	ActionUpdate    Action = "update"
	ActionDelete    Action = "delete"
	ActionUnchanged Action = "unchanged"
)

// GenerationResult represents a single generated file.
type GenerationResult struct {
	Path     string
	Content  []byte
	Existing []byte // nil if new file
	Action   Action
}
