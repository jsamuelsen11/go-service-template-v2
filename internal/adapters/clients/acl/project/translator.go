package project

import (
	"time"

	"github.com/jsamuelsen11/go-service-template-v2/internal/domain"
)

// ToDomainProject converts a downstream groupDTO to a domain Project entity.
// The downstream "Group" concept maps to our domain "Project" concept.
func ToDomainProject(dto groupDTO) domain.Project {
	createdAt, _ := time.Parse(time.RFC3339, dto.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, dto.UpdatedAt)

	return domain.Project{
		ID:          dto.ID,
		Name:        dto.Name,
		Description: dto.Description,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

// ToDomainProjectList converts a downstream groupListResponseDTO to a slice of
// domain Project entities.
func ToDomainProjectList(dto groupListResponseDTO) []domain.Project {
	projects := make([]domain.Project, len(dto.Groups))
	for i := range dto.Groups {
		projects[i] = ToDomainProject(dto.Groups[i])
	}
	return projects
}

// ToCreateGroupRequest converts a domain Project entity to a downstream
// createGroupRequestDTO.
func ToCreateGroupRequest(project *domain.Project) createGroupRequestDTO {
	return createGroupRequestDTO{
		Name:        project.Name,
		Description: project.Description,
	}
}

// ToUpdateGroupRequest converts a domain Project entity to a downstream
// updateGroupRequestDTO. All fields are set (full replacement semantics).
func ToUpdateGroupRequest(project *domain.Project) updateGroupRequestDTO {
	return updateGroupRequestDTO{
		Name:        &project.Name,
		Description: &project.Description,
	}
}
