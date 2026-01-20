package scheduler

import (
	"sort"

	"github.com/user/chore-scheduler/internal/models"
)

// CalculatePriority calculates the priority score for a task
// Higher scores indicate higher urgency
// Overdue tasks: 1000 + days overdue
// Due today: 1000
// Future tasks: 100 / days until due
// Unscheduled tasks: 50
func CalculatePriority(task *models.Task) float64 {
	if task.NextScheduled == nil {
		return 50 // Unscheduled tasks get medium priority
	}

	if task.IsOverdue() {
		return 1000 + float64(task.DaysOverdue())
	}

	daysUntil := task.DaysUntilDue()
	if daysUntil <= 0 {
		return 1000 // Due today
	}

	return 100.0 / float64(daysUntil)
}

// SortByPriority sorts tasks by priority in descending order (highest priority first)
func SortByPriority(tasks []*models.Task) []*models.Task {
	sorted := make([]*models.Task, len(tasks))
	copy(sorted, tasks)

	sort.Slice(sorted, func(i, j int) bool {
		return CalculatePriority(sorted[i]) > CalculatePriority(sorted[j])
	})

	return sorted
}
