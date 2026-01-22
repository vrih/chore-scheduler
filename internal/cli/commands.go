package cli

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/user/chore-scheduler/internal/models"
)

// buildAddCommand creates the add subcommand
func (c *CLI) buildAddCommand() *cobra.Command {
	var effort int
	var frequency int
	var room string

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a new task",
		Long:  `Add a new task to the scheduler with specified effort level, frequency, and room.`,
		Example: `  chore-scheduler add "Vacuum living room" --effort 2 --frequency 3 --room "Living Room"
  chore-scheduler add "Clean counters" --effort 1 --frequency 3 --room Kitchen`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if room == "" {
				return fmt.Errorf("room is required; use --room to specify the room")
			}
			return c.handleAdd(args[0], effort, frequency, room)
		},
	}

	cmd.Flags().IntVarP(&effort, "effort", "e", 2, "effort level (1=quick, 2=medium, 3=long)")
	cmd.Flags().IntVarP(&frequency, "frequency", "f", 7, "days between task occurrences")
	cmd.Flags().StringVarP(&room, "room", "r", "", "room this task belongs to (required)")

	return cmd
}

// buildListCommand creates the list subcommand
func (c *CLI) buildListCommand() *cobra.Command {
	var showOverdue bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all tasks",
		Long:  `Display all tasks with their effort, frequency, and next due date.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleList(showOverdue)
		},
	}

	cmd.Flags().BoolVar(&showOverdue, "overdue", false, "show only overdue tasks")

	return cmd
}

// buildUpdateCommand creates the update subcommand
func (c *CLI) buildUpdateCommand() *cobra.Command {
	var name string
	var effort int
	var frequency int
	var room string

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a task",
		Long:  `Update an existing task's name, effort, frequency, or room.`,
		Example: `  chore-scheduler update 1 --name "New name"
  chore-scheduler update 1 --effort 3 --frequency 14
  chore-scheduler update 1 --room "Living Room"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid task ID: %s", args[0])
			}
			return c.handleUpdate(id, name, effort, frequency, room)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "new task name")
	cmd.Flags().IntVarP(&effort, "effort", "e", 0, "new effort level (1-3)")
	cmd.Flags().IntVarP(&frequency, "frequency", "f", 0, "new frequency in days")
	cmd.Flags().StringVarP(&room, "room", "r", "", "new room assignment")

	return cmd
}

// buildDeleteCommand creates the delete subcommand
func (c *CLI) buildDeleteCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a task",
		Long:  `Delete a task and all its associated data.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid task ID: %s", args[0])
			}
			return c.handleDelete(id, force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation")

	return cmd
}

// buildTodayCommand creates the today subcommand
func (c *CLI) buildTodayCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "today",
		Short: "Show today's scheduled tasks",
		Long:  `Display all tasks scheduled for today.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleToday()
		},
	}
}

// buildUpcomingCommand creates the upcoming subcommand
func (c *CLI) buildUpcomingCommand() *cobra.Command {
	var days int

	cmd := &cobra.Command{
		Use:   "upcoming",
		Short: "Show upcoming scheduled tasks",
		Long:  `Display tasks scheduled for the upcoming days.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleUpcoming(days)
		},
	}

	cmd.Flags().IntVarP(&days, "days", "d", 7, "number of days to show")

	return cmd
}

// buildCompleteCommand creates the complete subcommand
func (c *CLI) buildCompleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "complete <id> [id...]",
		Short: "Mark one or more tasks as complete",
		Long:  `Mark one or more tasks as completed. The next occurrence will be scheduled automatically.`,
		Example: `  chore-scheduler complete 1
  chore-scheduler complete 1 2 3`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ids := make([]int64, 0, len(args))
			for _, arg := range args {
				id, err := strconv.ParseInt(arg, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid task ID: %s", arg)
				}
				ids = append(ids, id)
			}
			return c.handleCompleteMultiple(ids)
		},
	}
}

// buildPostponeCommand creates the postpone subcommand
func (c *CLI) buildPostponeCommand() *cobra.Command {
	var days int

	cmd := &cobra.Command{
		Use:   "postpone <id>",
		Short: "Postpone a task",
		Long:  `Postpone a task to the next available day.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid task ID: %s", args[0])
			}
			return c.handlePostpone(id, days)
		},
	}

	cmd.Flags().IntVarP(&days, "days", "d", 1, "minimum days to postpone")

	return cmd
}

// buildConfigCommand creates the config subcommand
func (c *CLI) buildConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  `Get, set, or list configuration values.`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleConfigList()
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleConfigGet(args[0])
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleConfigSet(args[0], args[1])
		},
	})

	return cmd
}

// buildRescheduleCommand creates the reschedule subcommand
func (c *CLI) buildRescheduleCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "reschedule",
		Short: "Reschedule all tasks",
		Long:  `Clear all schedules and reschedule all tasks from scratch.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleReschedule(force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation")

	return cmd
}

// buildRoomsCommand creates the rooms subcommand
func (c *CLI) buildRoomsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "rooms [room-name]",
		Short: "Show room cleanliness summary",
		Long: `Display cleanliness status of all rooms or tasks in a specific room.
Without arguments, shows a summary of all rooms.
With a room name, shows tasks in that room.`,
		Example: `  chore-scheduler rooms                  # Show all rooms summary
  chore-scheduler rooms Kitchen          # Show tasks in Kitchen`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return c.handleRoomsSummary()
			}
			return c.handleRoomTasks(args[0])
		},
	}
}

// Handler implementations

func (c *CLI) handleAdd(name string, effort, frequency int, room string) error {
	task := &models.Task{
		Name:          name,
		Room:          room,
		Effort:        effort,
		FrequencyDays: frequency,
	}

	if err := c.taskRepo.Create(task); err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	// Schedule the new task
	if err := c.scheduler.ScheduleTask(task); err != nil {
		return fmt.Errorf("failed to schedule task: %w", err)
	}

	// Get the scheduled date
	scheduled, err := c.scheduledRepo.GetByTask(task.ID)
	if err != nil {
		return fmt.Errorf("failed to get schedule: %w", err)
	}

	fmt.Printf("Created task #%d: %s\n", task.ID, task.Name)
	fmt.Printf("  Room: %s, Effort: %d, Frequency: every %d days\n", task.Room, task.Effort, task.FrequencyDays)
	if len(scheduled) > 0 {
		fmt.Printf("  Scheduled for: %s\n", scheduled[0].ScheduledDate.Format("Mon, Jan 2 2006"))
	}

	return nil
}

func (c *CLI) handleList(overdueOnly bool) error {
	var tasks []*models.Task
	var err error

	if overdueOnly {
		tasks, err = c.taskRepo.GetOverdue()
	} else {
		tasks, err = c.taskRepo.GetAll()
	}

	if err != nil {
		return fmt.Errorf("failed to get tasks: %w", err)
	}

	if len(tasks) == 0 {
		if overdueOnly {
			fmt.Println("No overdue tasks.")
		} else {
			fmt.Println("No tasks found. Add one with: chore-scheduler add \"Task name\" --room Room")
		}
		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("ID", "Name", "Room", "Effort", "Frequency", "Next Due", "Status", "Cleanliness")

	for _, task := range tasks {
		nextDue := "-"
		status := ""

		if task.NextScheduled != nil {
			nextDue = task.NextScheduled.Format("Jan 2")
			if task.IsOverdue() {
				status = "OVERDUE"
			} else if task.IsDueToday() {
				status = "TODAY"
			}
		}

		table.Append([]string{
			strconv.FormatInt(task.ID, 10),
			task.Name,
			task.Room,
			strconv.Itoa(task.Effort),
			fmt.Sprintf("%d days", task.FrequencyDays),
			nextDue,
			status,
			task.CleanlinessStatus(),
		})
	}

	table.Render()
	return nil
}

func (c *CLI) handleUpdate(id int64, name string, effort, frequency int, room string) error {
	task, err := c.taskRepo.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Apply updates
	if name != "" {
		task.Name = name
	}
	if room != "" {
		task.Room = room
	}
	if effort > 0 {
		task.Effort = effort
	}
	if frequency > 0 {
		task.FrequencyDays = frequency
	}

	if err := c.taskRepo.Update(task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Reschedule if effort or frequency changed
	if effort > 0 || frequency > 0 {
		if err := c.scheduler.ScheduleTask(task); err != nil {
			return fmt.Errorf("failed to reschedule task: %w", err)
		}
	}

	fmt.Printf("Updated task #%d: %s\n", task.ID, task.Name)
	return nil
}

func (c *CLI) handleDelete(id int64, force bool) error {
	task, err := c.taskRepo.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	if !force {
		fmt.Printf("Delete task #%d: %s? [y/N] ", task.ID, task.Name)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if err := c.taskRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	fmt.Printf("Deleted task #%d: %s\n", task.ID, task.Name)
	return nil
}

func (c *CLI) handleToday() error {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	scheduled, err := c.scheduledRepo.GetByDate(today)
	if err != nil {
		return fmt.Errorf("failed to get scheduled tasks: %w", err)
	}

	if len(scheduled) == 0 {
		fmt.Println("No tasks scheduled for today.")
		return nil
	}

	fmt.Printf("Tasks for today (%s):\n\n", today.Format("Mon, Jan 2 2006"))

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("ID", "Name", "Room", "Effort", "Status")

	totalEffort := 0
	for _, st := range scheduled {
		task, err := c.taskRepo.Get(st.TaskID)
		if err != nil {
			continue
		}

		status := ""
		if task.IsOverdue() {
			status = "OVERDUE"
		}

		table.Append([]string{
			strconv.FormatInt(task.ID, 10),
			task.Name,
			task.Room,
			strconv.Itoa(task.Effort),
			status,
		})
		totalEffort += task.Effort
	}

	table.Render()
	fmt.Printf("\nTotal effort: %d\n", totalEffort)

	return nil
}

func (c *CLI) handleUpcoming(days int) error {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	fmt.Printf("Upcoming tasks (%d days):\n\n", days)

	for i := 0; i < days; i++ {
		date := today.AddDate(0, 0, i)
		scheduled, err := c.scheduledRepo.GetByDate(date)
		if err != nil {
			return fmt.Errorf("failed to get scheduled tasks: %w", err)
		}

		if len(scheduled) == 0 {
			continue
		}

		dayLabel := date.Format("Mon, Jan 2")
		if i == 0 {
			dayLabel += " (today)"
		} else if i == 1 {
			dayLabel += " (tomorrow)"
		}

		fmt.Printf("%s:\n", dayLabel)

		totalEffort := 0
		for _, st := range scheduled {
			task, err := c.taskRepo.Get(st.TaskID)
			if err != nil {
				continue
			}
			fmt.Printf("  [%d] %s [%s] (effort: %d)\n", task.ID, task.Name, task.Room, task.Effort)
			totalEffort += task.Effort
		}
		fmt.Printf("  Total effort: %d\n\n", totalEffort)
	}

	return nil
}

func (c *CLI) handleComplete(id int64) error {
	task, err := c.taskRepo.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Record completion
	completion := models.NewCompletion(task.ID)
	if err := c.completionRepo.Create(completion); err != nil {
		return fmt.Errorf("failed to record completion: %w", err)
	}

	// Update task
	now := time.Now()
	task.LastCompleted = &now
	nextScheduled := task.CalculateNextScheduled()
	task.NextScheduled = &nextScheduled

	if err := c.taskRepo.Update(task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Clear current schedule and reschedule
	if err := c.scheduler.ClearSchedule(task.ID); err != nil {
		return fmt.Errorf("failed to clear schedule: %w", err)
	}

	if err := c.scheduler.ScheduleTask(task); err != nil {
		return fmt.Errorf("failed to reschedule task: %w", err)
	}

	// Get new scheduled date
	scheduled, err := c.scheduledRepo.GetByTask(task.ID)
	if err != nil {
		return fmt.Errorf("failed to get schedule: %w", err)
	}

	fmt.Printf("Completed: %s\n", task.Name)
	if len(scheduled) > 0 {
		fmt.Printf("Next scheduled: %s\n", scheduled[0].ScheduledDate.Format("Mon, Jan 2 2006"))
	}

	return nil
}

func (c *CLI) handleCompleteMultiple(ids []int64) error {
	var completed []string
	var failures []string

	for _, id := range ids {
		task, err := c.taskRepo.Get(id)
		if err != nil {
			failures = append(failures, fmt.Sprintf("#%d: task not found", id))
			continue
		}

		// Record completion
		completion := models.NewCompletion(task.ID)
		if err := c.completionRepo.Create(completion); err != nil {
			failures = append(failures, fmt.Sprintf("#%d (%s): failed to record completion", id, task.Name))
			continue
		}

		// Update task
		now := time.Now()
		task.LastCompleted = &now
		nextScheduled := task.CalculateNextScheduled()
		task.NextScheduled = &nextScheduled

		if err := c.taskRepo.Update(task); err != nil {
			failures = append(failures, fmt.Sprintf("#%d (%s): failed to update task", id, task.Name))
			continue
		}

		// Clear current schedule and reschedule other tasks if this was an early completion
		_, _, err = c.scheduler.CompleteTaskAndReschedule(task.ID)
		if err != nil {
			failures = append(failures, fmt.Sprintf("#%d (%s): failed to clear schedule", id, task.Name))
			continue
		}

		// Schedule the next occurrence of this task
		if err := c.scheduler.ScheduleTask(task); err != nil {
			failures = append(failures, fmt.Sprintf("#%d (%s): failed to reschedule", id, task.Name))
			continue
		}

		completed = append(completed, fmt.Sprintf("#%d: %s", task.ID, task.Name))
	}

	// Print results
	if len(completed) > 0 {
		fmt.Printf("Completed %d task(s):\n", len(completed))
		for _, c := range completed {
			fmt.Printf("  %s\n", c)
		}
	}

	if len(failures) > 0 {
		fmt.Printf("\nFailed to complete %d task(s):\n", len(failures))
		for _, f := range failures {
			fmt.Printf("  %s\n", f)
		}
	}

	// Return error only if all tasks failed
	if len(completed) == 0 && len(failures) > 0 {
		return fmt.Errorf("failed to complete any tasks")
	}

	return nil
}

func (c *CLI) handlePostpone(id int64, minDays int) error {
	task, err := c.taskRepo.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Clear current schedule
	if err := c.scheduler.ClearSchedule(task.ID); err != nil {
		return fmt.Errorf("failed to clear schedule: %w", err)
	}

	// Update next_scheduled to be at least minDays from now
	now := time.Now()
	minDate := now.AddDate(0, 0, minDays)
	task.NextScheduled = &minDate

	if err := c.taskRepo.Update(task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Reschedule
	if err := c.scheduler.ScheduleTask(task); err != nil {
		return fmt.Errorf("failed to reschedule task: %w", err)
	}

	// Get new scheduled date
	scheduled, err := c.scheduledRepo.GetByTask(task.ID)
	if err != nil {
		return fmt.Errorf("failed to get schedule: %w", err)
	}

	fmt.Printf("Postponed: %s\n", task.Name)
	if len(scheduled) > 0 {
		fmt.Printf("New scheduled date: %s\n", scheduled[0].ScheduledDate.Format("Mon, Jan 2 2006"))
	}

	return nil
}

func (c *CLI) handleConfigList() error {
	configs, err := c.configRepo.GetAll()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	if len(configs) == 0 {
		fmt.Println("No configuration values set.")
		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Key", "Value")

	for _, config := range configs {
		table.Append([]string{config.Key, config.Value})
	}

	table.Render()
	return nil
}

func (c *CLI) handleConfigGet(key string) error {
	value, err := c.configRepo.Get(key)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	fmt.Printf("%s = %s\n", key, value)
	return nil
}

func (c *CLI) handleConfigSet(key, value string) error {
	// Special handling for max_daily_effort
	if key == models.ConfigKeyMaxDailyEffort {
		effort, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid value for max_daily_effort: %s", value)
		}
		if err := c.configRepo.SetMaxDailyEffort(effort); err != nil {
			return fmt.Errorf("failed to set config: %w", err)
		}
	} else {
		if err := c.configRepo.Set(key, value); err != nil {
			return fmt.Errorf("failed to set config: %w", err)
		}
	}

	fmt.Printf("Set %s = %s\n", key, value)

	// Trigger reschedule if max effort changed
	if key == models.ConfigKeyMaxDailyEffort {
		fmt.Println("Rescheduling tasks with new effort limit...")
		if err := c.scheduler.Reschedule(); err != nil {
			return fmt.Errorf("failed to reschedule: %w", err)
		}
		fmt.Println("Done.")
	}

	return nil
}

func (c *CLI) handleReschedule(force bool) error {
	if !force {
		fmt.Print("This will clear all schedules and reschedule from scratch. Continue? [y/N] ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	fmt.Println("Rescheduling all tasks...")
	if err := c.scheduler.Reschedule(); err != nil {
		return fmt.Errorf("failed to reschedule: %w", err)
	}

	fmt.Println("Done.")
	return nil
}

func (c *CLI) handleRoomsSummary() error {
	rooms, err := c.taskRepo.GetAllRooms()
	if err != nil {
		return fmt.Errorf("failed to get rooms: %w", err)
	}

	if len(rooms) == 0 {
		fmt.Println("No rooms found. Add tasks with: chore-scheduler add \"Task name\" --room Room")
		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Room", "Clean", "Due", "Dirty", "Very Dirty", "Unknown", "Overall Status")

	for _, room := range rooms {
		tasks, err := c.taskRepo.GetByRoom(room)
		if err != nil {
			return fmt.Errorf("failed to get tasks for room %s: %w", room, err)
		}

		// Count cleanliness statuses
		counts := map[string]int{
			models.CleanlinessClean:     0,
			models.CleanlinessDue:       0,
			models.CleanlinessDirty:     0,
			models.CleanlinessVeryDirty: 0,
			models.CleanlinessUnknown:   0,
		}
		worstStatus := models.CleanlinessClean

		for _, task := range tasks {
			status := task.CleanlinessStatus()
			counts[status]++

			// Track worst status
			if isWorseThan(status, worstStatus) {
				worstStatus = status
			}
		}

		overallStatus := roomOverallStatus(worstStatus)

		table.Append([]string{
			room,
			strconv.Itoa(counts[models.CleanlinessClean]),
			strconv.Itoa(counts[models.CleanlinessDue]),
			strconv.Itoa(counts[models.CleanlinessDirty]),
			strconv.Itoa(counts[models.CleanlinessVeryDirty]),
			strconv.Itoa(counts[models.CleanlinessUnknown]),
			overallStatus,
		})
	}

	table.Render()
	return nil
}

func (c *CLI) handleRoomTasks(room string) error {
	tasks, err := c.taskRepo.GetByRoom(room)
	if err != nil {
		return fmt.Errorf("failed to get tasks: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Printf("No tasks found in room: %s\n", room)
		return nil
	}

	fmt.Printf("Tasks in %s:\n\n", room)

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("ID", "Name", "Effort", "Frequency", "Last Done", "Cleanliness")

	for _, task := range tasks {
		lastDone := "Never"
		if task.LastCompleted != nil {
			lastDone = task.LastCompleted.Format("Jan 2")
		}

		table.Append([]string{
			strconv.FormatInt(task.ID, 10),
			task.Name,
			strconv.Itoa(task.Effort),
			fmt.Sprintf("%d days", task.FrequencyDays),
			lastDone,
			task.CleanlinessStatus(),
		})
	}

	table.Render()
	return nil
}

// isWorseThan returns true if status a is worse than status b
func isWorseThan(a, b string) bool {
	order := map[string]int{
		models.CleanlinessClean:     0,
		models.CleanlinessUnknown:   1,
		models.CleanlinessDue:       2,
		models.CleanlinessDirty:     3,
		models.CleanlinessVeryDirty: 4,
	}
	return order[a] > order[b]
}

// roomOverallStatus returns a human-readable overall status for a room
func roomOverallStatus(worstStatus string) string {
	switch worstStatus {
	case models.CleanlinessClean:
		return "Spotless"
	case models.CleanlinessUnknown:
		return "Needs Review"
	case models.CleanlinessDue:
		return "Good"
	case models.CleanlinessDirty:
		return "Needs Attention"
	case models.CleanlinessVeryDirty:
		return "Urgent"
	default:
		return "Unknown"
	}
}
