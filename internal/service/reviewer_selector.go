package service

import (
	"math/rand"
	"time"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
)

// SelectReviewers выбирает до maxReviewers активных ревьюверов из команды, исключая excludeUserID
func SelectReviewers(teamMembers []*domain.User, excludeUserID string, maxReviewers int) []string {
	if maxReviewers <= 0 {
		return []string{}
	}

	candidates := make([]*domain.User, 0)
	for _, member := range teamMembers {
		if member.IsActive && member.ID != excludeUserID {
			candidates = append(candidates, member)
		}
	}

	if len(candidates) == 0 {
		return []string{}
	}

	count := len(candidates)
	if count > maxReviewers {
		count = maxReviewers
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	
	indices := make([]int, len(candidates))
	for i := range indices {
		indices[i] = i
	}
	
	for i := len(indices) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		indices[i], indices[j] = indices[j], indices[i]
	}
	
	selected := make([]string, 0, count)
	for i := 0; i < count && i < len(indices); i++ {
		selected = append(selected, candidates[indices[i]].ID)
	}

	return selected
}

