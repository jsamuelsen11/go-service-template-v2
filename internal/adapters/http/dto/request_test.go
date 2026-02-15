package dto_test

import (
	"errors"
	"testing"

	"github.com/jsamuelsen11/go-service-template-v2/internal/adapters/http/dto"
	"github.com/jsamuelsen11/go-service-template-v2/internal/domain"
)

func stringPtr(s string) *string { return &s }
func intPtr(i int) *int          { return &i }

// requireValidationField asserts err wraps ErrValidation and the resulting
// ValidationError contains the expected field key.
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

func TestCreateTodoRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		req       dto.CreateTodoRequest
		wantErr   bool
		wantField string
	}{
		{
			name: "valid request passes",
			req: dto.CreateTodoRequest{
				Title:       "Buy groceries",
				Description: "Milk, eggs, bread",
			},
			wantErr: false,
		},
		{
			name: "valid request with all fields",
			req: dto.CreateTodoRequest{
				Title:           "Buy groceries",
				Description:     "Milk, eggs, bread",
				Status:          "pending",
				Category:        "personal",
				ProgressPercent: 50,
			},
			wantErr: false,
		},
		{
			name: "empty title fails",
			req: dto.CreateTodoRequest{
				Title:       "",
				Description: "Some description",
			},
			wantErr:   true,
			wantField: "title",
		},
		{
			name: "whitespace-only title fails",
			req: dto.CreateTodoRequest{
				Title:       "   ",
				Description: "Some description",
			},
			wantErr:   true,
			wantField: "title",
		},
		{
			name: "empty description fails",
			req: dto.CreateTodoRequest{
				Title:       "Some title",
				Description: "",
			},
			wantErr:   true,
			wantField: "description",
		},
		{
			name: "invalid status fails",
			req: dto.CreateTodoRequest{
				Title:       "Some title",
				Description: "Some description",
				Status:      "completed",
			},
			wantErr:   true,
			wantField: "status",
		},
		{
			name: "valid status passes",
			req: dto.CreateTodoRequest{
				Title:       "Some title",
				Description: "Some description",
				Status:      "in_progress",
			},
			wantErr: false,
		},
		{
			name: "empty status passes (optional)",
			req: dto.CreateTodoRequest{
				Title:       "Some title",
				Description: "Some description",
				Status:      "",
			},
			wantErr: false,
		},
		{
			name: "invalid category fails",
			req: dto.CreateTodoRequest{
				Title:       "Some title",
				Description: "Some description",
				Category:    "urgent",
			},
			wantErr:   true,
			wantField: "category",
		},
		{
			name: "empty category passes (optional)",
			req: dto.CreateTodoRequest{
				Title:       "Some title",
				Description: "Some description",
				Category:    "",
			},
			wantErr: false,
		},
		{
			name: "negative progress fails",
			req: dto.CreateTodoRequest{
				Title:           "Some title",
				Description:     "Some description",
				ProgressPercent: -1,
			},
			wantErr:   true,
			wantField: "progress_percent",
		},
		{
			name: "progress over 100 fails",
			req: dto.CreateTodoRequest{
				Title:           "Some title",
				Description:     "Some description",
				ProgressPercent: 101,
			},
			wantErr:   true,
			wantField: "progress_percent",
		},
		{
			name: "progress boundary 0 passes",
			req: dto.CreateTodoRequest{
				Title:           "Some title",
				Description:     "Some description",
				ProgressPercent: 0,
			},
			wantErr: false,
		},
		{
			name: "progress boundary 100 passes",
			req: dto.CreateTodoRequest{
				Title:           "Some title",
				Description:     "Some description",
				ProgressPercent: 100,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.req.Validate()
			if tt.wantErr {
				requireValidationField(t, err, tt.wantField)
			} else if err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
		})
	}
}

func TestCreateTodoRequest_Validate_MultipleErrors(t *testing.T) {
	t.Parallel()

	req := dto.CreateTodoRequest{
		Title:           "",
		Description:     "",
		Status:          "bad",
		Category:        "bad",
		ProgressPercent: 200,
	}

	err := req.Validate()
	if err == nil {
		t.Fatal("Validate() = nil, want error with multiple failures")
	}

	var verr *domain.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("errors.As(err, *ValidationError) = false, got %T", err)
	}

	expectedFields := []string{"title", "description", "status", "category", "progress_percent"}
	for _, field := range expectedFields {
		if _, ok := verr.Fields[field]; !ok {
			t.Errorf("ValidationError.Fields missing key %q, got %v", field, verr.Fields)
		}
	}

	if len(verr.Fields) != len(expectedFields) {
		t.Errorf("ValidationError.Fields has %d entries, want %d", len(verr.Fields), len(expectedFields))
	}
}

func TestUpdateTodoRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		req       dto.UpdateTodoRequest
		wantErr   bool
		wantField string
	}{
		{
			name:    "all nil passes (no-op update)",
			req:     dto.UpdateTodoRequest{},
			wantErr: false,
		},
		{
			name:    "valid title passes",
			req:     dto.UpdateTodoRequest{Title: stringPtr("New title")},
			wantErr: false,
		},
		{
			name:      "empty title fails",
			req:       dto.UpdateTodoRequest{Title: stringPtr("")},
			wantErr:   true,
			wantField: "title",
		},
		{
			name:      "whitespace-only title fails",
			req:       dto.UpdateTodoRequest{Title: stringPtr("  ")},
			wantErr:   true,
			wantField: "title",
		},
		{
			name:    "valid description passes",
			req:     dto.UpdateTodoRequest{Description: stringPtr("New desc")},
			wantErr: false,
		},
		{
			name:      "empty description fails",
			req:       dto.UpdateTodoRequest{Description: stringPtr("")},
			wantErr:   true,
			wantField: "description",
		},
		{
			name:    "valid status passes",
			req:     dto.UpdateTodoRequest{Status: stringPtr("done")},
			wantErr: false,
		},
		{
			name:      "invalid status fails",
			req:       dto.UpdateTodoRequest{Status: stringPtr("bad")},
			wantErr:   true,
			wantField: "status",
		},
		{
			name:    "valid category passes",
			req:     dto.UpdateTodoRequest{Category: stringPtr("work")},
			wantErr: false,
		},
		{
			name:      "invalid category fails",
			req:       dto.UpdateTodoRequest{Category: stringPtr("bad")},
			wantErr:   true,
			wantField: "category",
		},
		{
			name:    "valid progress passes",
			req:     dto.UpdateTodoRequest{ProgressPercent: intPtr(50)},
			wantErr: false,
		},
		{
			name:      "progress over 100 fails",
			req:       dto.UpdateTodoRequest{ProgressPercent: intPtr(101)},
			wantErr:   true,
			wantField: "progress_percent",
		},
		{
			name:      "negative progress fails",
			req:       dto.UpdateTodoRequest{ProgressPercent: intPtr(-1)},
			wantErr:   true,
			wantField: "progress_percent",
		},
		{
			name:    "progress boundary 0 passes",
			req:     dto.UpdateTodoRequest{ProgressPercent: intPtr(0)},
			wantErr: false,
		},
		{
			name:    "progress boundary 100 passes",
			req:     dto.UpdateTodoRequest{ProgressPercent: intPtr(100)},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.req.Validate()
			if tt.wantErr {
				requireValidationField(t, err, tt.wantField)
			} else if err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
		})
	}
}
