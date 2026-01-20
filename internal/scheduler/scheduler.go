package scheduler

import (
	"fmt"
	"time"

	"github.com/user/chore-scheduler/internal/models"
	"github.com/user/chore-scheduler/internal/repository"
)

const (
	// SchedulingWindowDays is how far ahead to schedule tasks
	SchedulingWindowDays = 30
)

// Scheduler handles task scheduling logic
type Scheduler struct {
	taskRepo      repository.TaskRepository
	configRepo    repository.ConfigRepository
	scheduledRepo repository.ScheduledTaskRepository
}

// NewScheduler creates a new Scheduler instance
func NewScheduler(
	taskRepo repository.TaskRepository,
	configRepo repository.ConfigRepository,
	scheduledRepo repository.ScheduledTaskRepository,
) *Scheduler {
	return &Scheduler{
		taskRepo:      taskRepo,
		configRepo:    configRepo,
		scheduledRepo: scheduledRepo,
	}
}

// Schedule runs the main scheduling algorithm
// It assigns all unscheduled and due tasks to available days
func (s *Scheduler) Schedule() error {
	maxEffort, err := s.configRepo.GetMaxDailyEffort()
	if err != nil {
		return fmt.Errorf("failed to get max daily effort: %w", err)
	}

	// Get all tasks that need scheduling
	tasks, err := s.taskRepo.GetNeedingSchedule()
	if err != nil {
		return fmt.Errorf("failed to get tasks: %w", err)
	}

	if len(tasks) == 0 {
		return nil
	}

	// Sort by priority
	tasks = SortByPriority(tasks)

	// Track which tasks have been scheduled in this run
	scheduled := make(map[int64]bool)

	// Schedule each task
	for _, task := range tasks {
		if scheduled[task.ID] {
			continue
		}

		// Check if task already has a schedule entry for the future
		alreadyScheduled, err := s.hasScheduleEntry(task.ID)
		if err != nil {
			return fmt.Errorf("failed to check schedule: %w", err)
		}
		if alreadyScheduled {
			scheduled[task.ID] = true
			continue
		}

		// Find the best day for this task
		startDate := s.getSchedulingStartDate(task)
		date, err := s.FindNextAvailableDay(task.Effort, startDate, maxEffort)
		if err != nil {
			return fmt.Errorf("failed to find available day for task %d: %w", task.ID, err)
		}

		// Create scheduled task entry
		st := models.NewScheduledTask(task.ID, date)
		if err := s.scheduledRepo.Create(st); err != nil {
			return fmt.Errorf("failed to schedule task %d: %w", task.ID, err)
		}

		scheduled[task.ID] = true
	}

	return nil
}

// ScheduleTask schedules a single task to the next available day
func (s *Scheduler) ScheduleTask(task *models.Task) error {
	maxEffort, err := s.configRepo.GetMaxDailyEffort()
	if err != nil {
		return fmt.Errorf("failed to get max daily effort: %w", err)
	}

	// Clear existing schedules for this task
	if err := s.scheduledRepo.ClearForTask(task.ID); err != nil {
		return fmt.Errorf("failed to clear schedule: %w", err)
	}

	// Find next available day
	startDate := s.getSchedulingStartDate(task)
	date, err := s.FindNextAvailableDay(task.Effort, startDate, maxEffort)
	if err != nil {
		return fmt.Errorf("failed to find available day: %w", err)
	}

	// Create scheduled task entry
	st := models.NewScheduledTask(task.ID, date)
	if err := s.scheduledRepo.Create(st); err != nil {
		return fmt.Errorf("failed to create schedule: %w", err)
	}

	return nil
}

// GetDailyEffort returns the total effort allocated for a specific date
func (s *Scheduler) GetDailyEffort(date time.Time) (int, error) {
	return s.scheduledRepo.GetDailyEffort(date)
}

// FindNextAvailableDay finds the next day with enough capacity for the given effort
func (s *Scheduler) FindNextAvailableDay(effort int, startDate time.Time, maxEffort int) (time.Time, error) {
	// Normalize start date to midnight
	date := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)

	for i := 0; i < SchedulingWindowDays; i++ {
		currentEffort, err := s.scheduledRepo.GetDailyEffort(date)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to get daily effort: %w", err)
		}

		remainingCapacity := maxEffort - currentEffort
		if remainingCapacity >= effort {
			return date, nil
		}

		date = date.AddDate(0, 0, 1)
	}

	// If no day found within window, return the last day checked
	// This ensures tasks always get scheduled eventually
	return date.AddDate(0, 0, -1), nil
}

// ClearSchedule removes all scheduled entries for a task
func (s *Scheduler) ClearSchedule(taskID int64) error {
	return s.scheduledRepo.ClearForTask(taskID)
}

// Reschedule clears all schedules and reschedules everything
func (s *Scheduler) Reschedule() error {
	if err := s.scheduledRepo.ClearAll(); err != nil {
		return fmt.Errorf("failed to clear schedules: %w", err)
	}
	return s.Schedule()
}

// getSchedulingStartDate determines when to start looking for available days
func (s *Scheduler) getSchedulingStartDate(task *models.Task) time.Time {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// If task has a next_scheduled date, use that as the starting point
	// unless it's in the past (overdue), then use today
	if task.NextScheduled != nil {
		scheduled := time.Date(
			task.NextScheduled.Year(),
			task.NextScheduled.Month(),
			task.NextScheduled.Day(),
			0, 0, 0, 0, time.UTC,
		)
		if scheduled.Before(today) {
			return today
		}
		return scheduled
	}

	return today
}

// hasScheduleEntry checks if a task already has a schedule entry
func (s *Scheduler) hasScheduleEntry(taskID int64) (bool, error) {
	entries, err := s.scheduledRepo.GetByTask(taskID)
	if err != nil {
		return false, err
	}

	// Check if any entry is for today or in the future
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	for _, entry := range entries {
		entryDate := time.Date(
			entry.ScheduledDate.Year(),
			entry.ScheduledDate.Month(),
			entry.ScheduledDate.Day(),
			0, 0, 0, 0, time.UTC,
		)
		if !entryDate.Before(today) {
			return true, nil
		}
	}

	return false, nil
}
