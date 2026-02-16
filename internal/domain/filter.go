package domain

// TodoFilter holds optional filter criteria for listing todos.
// Zero-value fields mean "no filter" for that dimension.
type TodoFilter struct {
	Status    TodoStatus
	Category  TodoCategory
	ProjectID *int64
}
