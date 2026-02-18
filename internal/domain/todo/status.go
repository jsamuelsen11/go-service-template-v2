package todo

// Status represents the completion state of a Todo.
type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

// IsValid returns true if the status is one of the defined constants.
func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusInProgress, StatusDone:
		return true
	default:
		return false
	}
}

// String implements fmt.Stringer.
func (s Status) String() string {
	return string(s)
}
