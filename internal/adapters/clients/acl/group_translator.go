package acl

import (
	"time"

	"github.com/jsamuelsen11/go-service-template-v2/internal/domain"
)

// toDomainProject converts a downstream groupDTO to a domain Project entity.
// The downstream "Group" concept maps to our domain "Project" concept.
func toDomainProject(dto groupDTO) domain.Project {
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

// toDomainProjectList converts a downstream groupListResponseDTO to a slice of
// domain Project entities.
func toDomainProjectList(dto groupListResponseDTO) []domain.Project {
	projects := make([]domain.Project, len(dto.Groups))
	for i := range dto.Groups {
		projects[i] = toDomainProject(dto.Groups[i])
	}
	return projects
}

// toCreateGroupRequest converts a domain Project entity to a downstream
// createGroupRequestDTO.
func toCreateGroupRequest(project *domain.Project) createGroupRequestDTO {
	return createGroupRequestDTO{
		Name:        project.Name,
		Description: project.Description,
	}
}

// toUpdateGroupRequest converts a domain Project entity to a downstream
// updateGroupRequestDTO. All fields are set (full replacement semantics).
func toUpdateGroupRequest(project *domain.Project) updateGroupRequestDTO {
	return updateGroupRequestDTO{
		Name:        &project.Name,
		Description: &project.Description,
	}
}
