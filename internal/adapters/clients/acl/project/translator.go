package project

import (
	"time"

	domproject "github.com/jsamuelsen11/go-service-template-v2/internal/domain/project"
)

// ToDomainProject converts a downstream GroupDTO to a domain Project entity.
// The downstream "Group" concept maps to our domain "Project" concept.
func ToDomainProject(dto GroupDTO) domproject.Project {
	createdAt, _ := time.Parse(time.RFC3339, dto.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, dto.UpdatedAt)

	return domproject.Project{
		ID:          dto.ID,
		Name:        dto.Name,
		Description: dto.Description,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

// ToDomainProjectList converts a downstream GroupListResponseDTO to a slice of
// domain Project entities.
func ToDomainProjectList(dto GroupListResponseDTO) []domproject.Project {
	projects := make([]domproject.Project, len(dto.Groups))
	for i := range dto.Groups {
		projects[i] = ToDomainProject(dto.Groups[i])
	}
	return projects
}

// ToCreateGroupRequest converts a domain Project entity to a downstream
// CreateGroupRequestDTO.
func ToCreateGroupRequest(project *domproject.Project) CreateGroupRequestDTO {
	return CreateGroupRequestDTO{
		Name:        project.Name,
		Description: project.Description,
	}
}

// ToUpdateGroupRequest converts a domain Project entity to a downstream
// UpdateGroupRequestDTO. All fields are set (full replacement semantics).
func ToUpdateGroupRequest(project *domproject.Project) UpdateGroupRequestDTO {
	return UpdateGroupRequestDTO{
		Name:        &project.Name,
		Description: &project.Description,
	}
}
