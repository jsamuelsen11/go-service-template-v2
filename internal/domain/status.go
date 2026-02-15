package domain

// TodoStatus represents the completion state of a Todo.
type TodoStatus string

const (
	StatusPending    TodoStatus = "pending"
	StatusInProgress TodoStatus = "in_progress"
	StatusDone       TodoStatus = "done"
)

// IsValid returns true if the status is one of the defined constants.
func (s TodoStatus) IsValid() bool {
	switch s {
	case StatusPending, StatusInProgress, StatusDone:
		return true
	default:
		return false
	}
}

// String implements fmt.Stringer.
func (s TodoStatus) String() string {
	return string(s)
}
