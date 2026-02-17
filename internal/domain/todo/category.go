package todo

// Category represents the categorization of a Todo item.
type Category string

const (
	CategoryPersonal Category = "personal"
	CategoryWork     Category = "work"
	CategoryOther    Category = "other"
)

// IsValid returns true if the category is one of the defined constants.
func (c Category) IsValid() bool {
	switch c {
	case CategoryPersonal, CategoryWork, CategoryOther:
		return true
	default:
		return false
	}
}

// String implements fmt.Stringer.
func (c Category) String() string {
	return string(c)
}
