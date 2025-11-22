package domain

import "fmt"

type DomainError struct {
	Code    string
	Message string
}

func (e *DomainError) Error() string {
	return e.Message
}

// Это позволяет использовать errors.Is()
func (e *DomainError) Is(target error) bool {
	if t, ok := target.(*DomainError); ok {
		return e.Code == t.Code
	}
	return false
}

var (
	// ErrTeamExists - команда уже существует
	ErrTeamExists = &DomainError{
		Code:    "TEAM_EXISTS",
		Message: "team_name already exists",
	}

	// ErrPRExists - PR уже существует
	ErrPRExists = &DomainError{
		Code:    "PR_EXISTS",
		Message: "PR id already exists",
	}

	// ErrPRMerged - нельзя изменять PR после merge
	ErrPRMerged = &DomainError{
		Code:    "PR_MERGED",
		Message: "cannot reassign on merged PR",
	}

	// ErrNotAssigned - ревьювер не назначен на PR
	ErrNotAssigned = &DomainError{
		Code:    "NOT_ASSIGNED",
		Message: "reviewer is not assigned to this PR",
	}

	// ErrNoCandidate - нет доступных кандидатов для замены
	ErrNoCandidate = &DomainError{
		Code:    "NO_CANDIDATE",
		Message: "no active replacement candidate in team",
	}

	// ErrNotFound - ресурс не найден
	ErrNotFound = &DomainError{
		Code:    "NOT_FOUND",
		Message: "resource not found",
	}
)

// NewNotFoundError создает ошибку NOT_FOUND с дополнительным контекстом
func NewNotFoundError(resource string) *DomainError {
	return &DomainError{
		Code:    "NOT_FOUND",
		Message: fmt.Sprintf("%s not found", resource),
	}
}
