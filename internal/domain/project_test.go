package domain

import (
	"errors"
	"testing"
	"time"
)

func validProject() Project {
	return Project{
		ID:          1,
		Name:        "Sprint 1",
		Description: "First sprint tasks",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func TestProject_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		modify    func(*Project)
		wantErr   bool
		wantField string
	}{
		{
			name:    "valid project passes",
			modify:  func(_ *Project) {},
			wantErr: false,
		},
		{
			name:      "empty name fails",
			modify:    func(p *Project) { p.Name = "" },
			wantErr:   true,
			wantField: "name",
		},
		{
			name:      "whitespace-only name fails",
			modify:    func(p *Project) { p.Name = "   " },
			wantErr:   true,
			wantField: "name",
		},
		{
			name:      "empty description fails",
			modify:    func(p *Project) { p.Description = "" },
			wantErr:   true,
			wantField: "description",
		},
		{
			name:      "whitespace-only description fails",
			modify:    func(p *Project) { p.Description = "\t\n" },
			wantErr:   true,
			wantField: "description",
		},
		{
			name:    "nil todos passes",
			modify:  func(p *Project) { p.Todos = nil },
			wantErr: false,
		},
		{
			name:    "empty todos passes",
			modify:  func(p *Project) { p.Todos = []Todo{} },
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := validProject()
			tt.modify(&p)
			err := p.Validate()

			if tt.wantErr {
				requireValidationField(t, err, tt.wantField)
			} else if err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
		})
	}
}

func TestProject_Validate_MultipleErrors(t *testing.T) {
	t.Parallel()

	p := Project{
		Name:        "",
		Description: "",
	}

	err := p.Validate()
	if err == nil {
		t.Fatal("Validate() = nil, want error with multiple failures")
	}

	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("errors.As(err, *ValidationError) = false, got %T", err)
	}

	expectedFields := []string{"name", "description"}
	for _, field := range expectedFields {
		if _, ok := verr.Fields[field]; !ok {
			t.Errorf("ValidationError.Fields missing key %q, got %v", field, verr.Fields)
		}
	}

	if len(verr.Fields) != len(expectedFields) {
		t.Errorf("ValidationError.Fields has %d entries, want %d", len(verr.Fields), len(expectedFields))
	}
}
