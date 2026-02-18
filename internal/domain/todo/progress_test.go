package todo

import "testing"

func TestCalculateProjectProgress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		todos []Todo
		want  int
	}{
		{
			name:  "empty slice returns 0",
			todos: []Todo{},
			want:  0,
		},
		{
			name:  "nil slice returns 0",
			todos: nil,
			want:  0,
		},
		{
			name:  "single todo at 0%",
			todos: []Todo{{ProgressPercent: 0}},
			want:  0,
		},
		{
			name:  "single todo at 100%",
			todos: []Todo{{ProgressPercent: 100}},
			want:  100,
		},
		{
			name:  "single todo at 50%",
			todos: []Todo{{ProgressPercent: 50}},
			want:  50,
		},
		{
			name: "two todos even average",
			todos: []Todo{
				{ProgressPercent: 0},
				{ProgressPercent: 100},
			},
			want: 50,
		},
		{
			name: "two todos uneven average truncates",
			todos: []Todo{
				{ProgressPercent: 33},
				{ProgressPercent: 66},
			},
			want: 49,
		},
		{
			name: "all complete",
			todos: []Todo{
				{ProgressPercent: 100},
				{ProgressPercent: 100},
				{ProgressPercent: 100},
			},
			want: 100,
		},
		{
			name: "all at zero",
			todos: []Todo{
				{ProgressPercent: 0},
				{ProgressPercent: 0},
				{ProgressPercent: 0},
			},
			want: 0,
		},
		{
			name: "mixed progress values",
			todos: []Todo{
				{ProgressPercent: 10},
				{ProgressPercent: 20},
				{ProgressPercent: 30},
				{ProgressPercent: 40},
			},
			want: 25,
		},
		{
			name: "integer truncation toward zero",
			todos: []Todo{
				{ProgressPercent: 1},
				{ProgressPercent: 1},
				{ProgressPercent: 1},
			},
			want: 1,
		},
		{
			name: "single percent divided by three truncates to zero",
			todos: []Todo{
				{ProgressPercent: 1},
				{ProgressPercent: 0},
				{ProgressPercent: 0},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := CalculateProjectProgress(tt.todos); got != tt.want {
				t.Errorf("CalculateProjectProgress() = %d, want %d", got, tt.want)
			}
		})
	}
}
