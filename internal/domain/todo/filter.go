package todo

// Filter holds optional filter criteria for listing todos.
// Zero-value fields mean "no filter" for that dimension.
type Filter struct {
	Status    Status
	Category  Category
	ProjectID *int64
}
