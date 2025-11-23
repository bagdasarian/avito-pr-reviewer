package service

import (
	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
	"math/rand"
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

	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	selected := make([]string, 0, count)
	for i := 0; i < count; i++ {
		selected = append(selected, candidates[i].ID)
	}

	return selected
}
