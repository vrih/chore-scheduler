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

// RescheduleFromDate reschedules all tasks from a given date onwards
// This is used to optimize the schedule after an early completion frees up capacity
func (s *Scheduler) RescheduleFromDate(fromDate time.Time) error {
	// Normalize date and use today if fromDate is in the past
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	fromDate = time.Date(fromDate.Year(), fromDate.Month(), fromDate.Day(), 0, 0, 0, 0, time.UTC)

	if fromDate.Before(today) {
		fromDate = today
	}

	// Get all scheduled entries from that date forward
	scheduled, err := s.scheduledRepo.GetFromDate(fromDate)
	if err != nil {
		return fmt.Errorf("failed to get scheduled tasks: %w", err)
	}

	if len(scheduled) == 0 {
		return nil
	}

	// Collect unique task IDs
	taskIDs := make(map[int64]bool)
	for _, st := range scheduled {
		taskIDs[st.TaskID] = true
	}

	// Fetch the actual tasks
	var tasks []*models.Task
	for taskID := range taskIDs {
		task, err := s.taskRepo.Get(taskID)
		if err != nil {
			continue // Task may have been deleted
		}
		tasks = append(tasks, task)
	}

	// Delete the schedule entries for these tasks
	for taskID := range taskIDs {
		if err := s.scheduledRepo.ClearForTask(taskID); err != nil {
			return fmt.Errorf("failed to clear schedule for task %d: %w", taskID, err)
		}
	}

	// Get max effort
	maxEffort, err := s.configRepo.GetMaxDailyEffort()
	if err != nil {
		return fmt.Errorf("failed to get max daily effort: %w", err)
	}

	// Sort tasks by priority
	tasks = SortByPriority(tasks)

	// Reschedule each task
	for _, task := range tasks {
		startDate := s.getSchedulingStartDate(task)
		// Use the later of fromDate or the task's natural start date
		if startDate.Before(fromDate) {
			startDate = fromDate
		}

		date, err := s.FindNextAvailableDay(task.Effort, startDate, maxEffort)
		if err != nil {
			return fmt.Errorf("failed to find available day for task %d: %w", task.ID, err)
		}

		st := models.NewScheduledTask(task.ID, date)
		if err := s.scheduledRepo.Create(st); err != nil {
			return fmt.Errorf("failed to schedule task %d: %w", task.ID, err)
		}
	}

	return nil
}

// CompleteTaskAndReschedule handles completing a task and rescheduling affected tasks
// Returns the original scheduled date (if any), whether it was an early completion, and any error
func (s *Scheduler) CompleteTaskAndReschedule(taskID int64) (originalDate *time.Time, wasEarly bool, err error) {
	// Get current schedule BEFORE clearing
	scheduled, err := s.scheduledRepo.GetByTask(taskID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get task schedule: %w", err)
	}

	// Find the future scheduled date (if any)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	var futureDate *time.Time
	for _, st := range scheduled {
		schedDate := time.Date(st.ScheduledDate.Year(), st.ScheduledDate.Month(), st.ScheduledDate.Day(), 0, 0, 0, 0, time.UTC)
		if schedDate.After(today) {
			futureDate = &schedDate
			break
		}
	}

	// Clear the schedule
	if err := s.ClearSchedule(taskID); err != nil {
		return nil, false, fmt.Errorf("failed to clear schedule: %w", err)
	}

	// If scheduled date was in future, this is an early completion
	if futureDate != nil {
		// Reschedule tasks from the original date to fill freed capacity
		if err := s.RescheduleFromDate(*futureDate); err != nil {
			return futureDate, true, fmt.Errorf("failed to reschedule from date: %w", err)
		}
		return futureDate, true, nil
	}

	return nil, false, nil
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
