package todo

// CalculateProjectProgress returns the average progress percentage across
// all provided todos. Returns 0 if the slice is empty.
func CalculateProjectProgress(todos []Todo) int {
	if len(todos) == 0 {
		return 0
	}
	var total int
	for i := range todos {
		total += todos[i].ProgressPercent
	}
	return total / len(todos)
}
