package todo

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jsamuelsen11/go-service-template-v2/internal/domain"
)

func int64Ptr(v int64) *int64 { return &v }

// requireValidationField is a test helper that asserts err wraps domain.ErrValidation
// and the resulting ValidationError contains the expected field key.
func requireValidationField(t *testing.T, err error, field string) {
	t.Helper()

	if err == nil {
		t.Fatal("Validate() = nil, want error")
	}
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("errors.Is(err, ErrValidation) = false, got %v", err)
	}

	var verr *domain.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("errors.As(err, *ValidationError) = false, got %T", err)
	}
	if _, ok := verr.Fields[field]; !ok {
		t.Errorf("ValidationError.Fields missing key %q, got %v", field, verr.Fields)
	}
}

func TestStatus_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status Status
		want   bool
	}{
		{
			name:   "pending is valid",
			status: StatusPending,
			want:   true,
		},
		{
			name:   "in_progress is valid",
			status: StatusInProgress,
			want:   true,
		},
		{
			name:   "done is valid",
			status: StatusDone,
			want:   true,
		},
		{
			name:   "empty string is invalid",
			status: "",
			want:   false,
		},
		{
			name:   "unknown value is invalid",
			status: "completed",
			want:   false,
		},
		{
			name:   "case sensitive",
			status: "Pending",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("Status(%q).IsValid() = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestStatus_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status Status
		want   string
	}{
		{StatusPending, "pending"},
		{StatusInProgress, "in_progress"},
		{StatusDone, "done"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if got := tt.status.String(); got != tt.want {
				t.Errorf("Status.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCategory_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		category Category
		want     bool
	}{
		{
			name:     "personal is valid",
			category: CategoryPersonal,
			want:     true,
		},
		{
			name:     "work is valid",
			category: CategoryWork,
			want:     true,
		},
		{
			name:     "other is valid",
			category: CategoryOther,
			want:     true,
		},
		{
			name:     "empty string is invalid",
			category: "",
			want:     false,
		},
		{
			name:     "unknown value is invalid",
			category: "hobby",
			want:     false,
		},
		{
			name:     "case sensitive",
			category: "Personal",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.category.IsValid(); got != tt.want {
				t.Errorf("Category(%q).IsValid() = %v, want %v", tt.category, got, tt.want)
			}
		})
	}
}

func TestCategory_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		category Category
		want     string
	}{
		{CategoryPersonal, "personal"},
		{CategoryWork, "work"},
		{CategoryOther, "other"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if got := tt.category.String(); got != tt.want {
				t.Errorf("Category.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func validTodo() Todo {
	return Todo{
		ID:              1,
		Title:           "Buy groceries",
		Description:     "Milk, eggs, bread",
		Status:          StatusPending,
		Category:        CategoryPersonal,
		ProgressPercent: 0,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

func TestTodo_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		modify    func(*Todo)
		wantErr   bool
		wantField string
	}{
		{
			name:    "valid todo passes",
			modify:  func(_ *Todo) {},
			wantErr: false,
		},
		{
			name:      "empty title fails",
			modify:    func(td *Todo) { td.Title = "" },
			wantErr:   true,
			wantField: "title",
		},
		{
			name:      "whitespace-only title fails",
			modify:    func(td *Todo) { td.Title = "   " },
			wantErr:   true,
			wantField: "title",
		},
		{
			name:      "empty description fails",
			modify:    func(td *Todo) { td.Description = "" },
			wantErr:   true,
			wantField: "description",
		},
		{
			name:      "whitespace-only description fails",
			modify:    func(td *Todo) { td.Description = "\t\n" },
			wantErr:   true,
			wantField: "description",
		},
		{
			name:      "invalid status fails",
			modify:    func(td *Todo) { td.Status = "completed" },
			wantErr:   true,
			wantField: "status",
		},
		{
			name:      "empty status fails",
			modify:    func(td *Todo) { td.Status = "" },
			wantErr:   true,
			wantField: "status",
		},
		{
			name:      "invalid category fails",
			modify:    func(td *Todo) { td.Category = "urgent" },
			wantErr:   true,
			wantField: "category",
		},
		{
			name:      "negative progress fails",
			modify:    func(td *Todo) { td.ProgressPercent = -1 },
			wantErr:   true,
			wantField: "progress_percent",
		},
		{
			name:      "progress over 100 fails",
			modify:    func(td *Todo) { td.ProgressPercent = 101 },
			wantErr:   true,
			wantField: "progress_percent",
		},
		{
			name:    "progress at boundary 0 passes",
			modify:  func(td *Todo) { td.ProgressPercent = 0 },
			wantErr: false,
		},
		{
			name: "progress at boundary 100 passes",
			modify: func(td *Todo) {
				td.ProgressPercent = 100
				td.Status = StatusDone
			},
			wantErr: false,
		},
		{
			name:    "all valid statuses accepted",
			modify:  func(td *Todo) { td.Status = StatusInProgress },
			wantErr: false,
		},
		{
			name:    "all valid categories accepted",
			modify:  func(td *Todo) { td.Category = CategoryWork },
			wantErr: false,
		},
		{
			name:    "nil project ID passes (ungrouped)",
			modify:  func(td *Todo) { td.ProjectID = nil },
			wantErr: false,
		},
		{
			name:    "positive project ID passes",
			modify:  func(td *Todo) { td.ProjectID = int64Ptr(1) },
			wantErr: false,
		},
		{
			name:      "zero project ID fails",
			modify:    func(td *Todo) { td.ProjectID = int64Ptr(0) },
			wantErr:   true,
			wantField: "project_id",
		},
		{
			name:      "negative project ID fails",
			modify:    func(td *Todo) { td.ProjectID = int64Ptr(-5) },
			wantErr:   true,
			wantField: "project_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			td := validTodo()
			tt.modify(&td)
			err := td.Validate()

			if tt.wantErr {
				requireValidationField(t, err, tt.wantField)
			} else if err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
		})
	}
}

func TestTodo_Validate_MultipleErrors(t *testing.T) {
	t.Parallel()

	td := Todo{
		Title:           "",
		Description:     "",
		Status:          "bad",
		Category:        "bad",
		ProgressPercent: 200,
		ProjectID:       int64Ptr(0),
	}

	err := td.Validate()
	if err == nil {
		t.Fatal("Validate() = nil, want error with multiple failures")
	}

	var verr *domain.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("errors.As(err, *ValidationError) = false, got %T", err)
	}

	expectedFields := []string{"title", "description", "status", "category", "progress_percent", "project_id"}
	for _, field := range expectedFields {
		if _, ok := verr.Fields[field]; !ok {
			t.Errorf("ValidationError.Fields missing key %q, got %v", field, verr.Fields)
		}
	}

	if len(verr.Fields) != len(expectedFields) {
		t.Errorf("ValidationError.Fields has %d entries, want %d", len(verr.Fields), len(expectedFields))
	}
}

func TestValidationError_ErrorsIs(t *testing.T) {
	t.Parallel()

	verr := &domain.ValidationError{Fields: map[string]string{"title": domain.MsgRequired}}

	if !errors.Is(verr, domain.ErrValidation) {
		t.Error("errors.Is(ValidationError, ErrValidation) = false, want true")
	}

	// Wrapped further
	wrapped := fmt.Errorf("operation failed: %w", verr)
	if !errors.Is(wrapped, domain.ErrValidation) {
		t.Error("errors.Is(wrapped ValidationError, ErrValidation) = false, want true")
	}
}

func TestValidationError_ErrorsAs(t *testing.T) {
	t.Parallel()

	original := &domain.ValidationError{Fields: map[string]string{
		"title":       domain.MsgRequired,
		"description": domain.MsgRequired,
	}}

	wrapped := fmt.Errorf("operation failed: %w", original)

	var verr *domain.ValidationError
	if !errors.As(wrapped, &verr) {
		t.Fatal("errors.As(wrapped, *ValidationError) = false, want true")
	}

	if len(verr.Fields) != 2 {
		t.Errorf("ValidationError.Fields has %d entries, want 2", len(verr.Fields))
	}
	if verr.Fields["title"] != domain.MsgRequired {
		t.Errorf("Fields[\"title\"] = %q, want %q", verr.Fields["title"], domain.MsgRequired)
	}
}

func TestValidationError_Error(t *testing.T) {
	t.Parallel()

	verr := &domain.ValidationError{Fields: map[string]string{"title": domain.MsgRequired}}
	got := verr.Error()

	if got == "" {
		t.Fatal("ValidationError.Error() returned empty string")
	}
	// Should contain the sentinel message prefix
	if !errors.Is(verr, domain.ErrValidation) {
		t.Error("should wrap ErrValidation")
	}
}

func TestSentinelErrors(t *testing.T) {
	t.Parallel()

	sentinels := []struct {
		name string
		err  error
	}{
		{"ErrNotFound", domain.ErrNotFound},
		{"ErrValidation", domain.ErrValidation},
		{"ErrConflict", domain.ErrConflict},
		{"ErrForbidden", domain.ErrForbidden},
		{"ErrUnavailable", domain.ErrUnavailable},
	}

	for _, tt := range sentinels {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Wrapping preserves identity
			wrapped := fmt.Errorf("context: %w", tt.err)
			if !errors.Is(wrapped, tt.err) {
				t.Errorf("errors.Is(wrapped, %s) = false", tt.name)
			}
		})
	}

	// All sentinels are distinct
	for i, a := range sentinels {
		for j, b := range sentinels {
			if i != j && errors.Is(a.err, b.err) {
				t.Errorf("%s and %s should be distinct", a.name, b.name)
			}
		}
	}
}
