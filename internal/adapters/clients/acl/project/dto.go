// Package project implements the Anti-Corruption Layer translators for the
// downstream TODO API's group resources, which map to domain Projects.
package project

// GroupDTO matches the downstream Group schema.
type GroupDTO struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// CreateGroupRequestDTO matches the downstream CreateGroupRequest schema.
type CreateGroupRequestDTO struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// UpdateGroupRequestDTO matches the downstream UpdateGroupRequest schema.
// All fields are optional; nil means "do not change this field.".
type UpdateGroupRequestDTO struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// GroupListResponseDTO matches the downstream GroupListResponse schema.
type GroupListResponseDTO struct {
	Groups []GroupDTO `json:"groups"`
	Count  int64      `json:"count"`
}
