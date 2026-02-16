// Package project implements the Anti-Corruption Layer translators for the
// downstream TODO API's group resources, which map to domain Projects.
package project

// groupDTO matches the downstream Group schema.
type groupDTO struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// createGroupRequestDTO matches the downstream CreateGroupRequest schema.
type createGroupRequestDTO struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// updateGroupRequestDTO matches the downstream UpdateGroupRequest schema.
// All fields are optional; nil means "do not change this field.".
type updateGroupRequestDTO struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// groupListResponseDTO matches the downstream GroupListResponse schema.
type groupListResponseDTO struct {
	Groups []groupDTO `json:"groups"`
	Count  int64      `json:"count"`
}
