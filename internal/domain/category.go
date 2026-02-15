package domain

// TodoCategory represents the categorization of a Todo item.
type TodoCategory string

const (
	CategoryPersonal TodoCategory = "personal"
	CategoryWork     TodoCategory = "work"
	CategoryOther    TodoCategory = "other"
)

// IsValid returns true if the category is one of the defined constants.
func (c TodoCategory) IsValid() bool {
	switch c {
	case CategoryPersonal, CategoryWork, CategoryOther:
		return true
	default:
		return false
	}
}

// String implements fmt.Stringer.
func (c TodoCategory) String() string {
	return string(c)
}
