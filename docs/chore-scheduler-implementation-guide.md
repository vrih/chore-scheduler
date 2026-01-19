# Chore Scheduler - Implementation Guide

## Development Phases

### Phase 1: Project Setup and Database Layer
**Goal**: Create project structure and database foundation

#### Tasks:
1. Initialize Go module
   ```bash
   go mod init github.com/user/chore-scheduler
   ```

2. Create directory structure
   ```bash
   mkdir -p cmd/chore-scheduler
   mkdir -p internal/{db,models,repository,scheduler,cli}
   ```

3. Add dependencies
   ```bash
   go get github.com/mattn/go-sqlite3
   go get github.com/spf13/cobra
   go get github.com/stretchr/testify
   go get github.com/olekukonko/tablewriter
   ```

4. Implement `internal/db/db.go`
   - Database connection management
   - Connection pool configuration
   - Close method

5. Implement `internal/db/migrations.go`
   - Create tables schema
   - Add indexes
   - Insert default config
   - Migration versioning (future-proof)

6. Write tests for database initialization
   - Test schema creation
   - Test default values
   - Test in-memory database

**Validation**: Database can be created and initialized with proper schema

---

### Phase 2: Models
**Goal**: Define core data structures

#### Tasks:
1. Implement `internal/models/task.go`
   - Task struct
   - Helper methods: `DaysUntilDue()`, `IsOverdue()`, `DaysOverdue()`
   - Validation methods

2. Implement `internal/models/completion.go`
   - Completion struct
   - Basic validation

3. Implement `internal/models/config.go`
   - Config struct
   - Typed accessors for known config values

4. Implement `internal/models/scheduled_task.go`
   - ScheduledTask struct for daily assignments

5. Write model tests
   - Test date calculations
   - Test validation logic
   - Test edge cases (nil dates, etc.)

**Validation**: Models correctly represent domain concepts and handle dates properly

---

### Phase 3: Repository Layer
**Goal**: Implement data access layer

#### Tasks:
1. Implement `internal/repository/task_repository.go`
   - Create, Get, GetAll, Update, Delete
   - GetByScheduledDate, GetUnscheduled, GetOverdue
   - Proper error handling and SQL injection prevention

2. Implement `internal/repository/completion_repository.go`
   - Create, GetByTaskID, GetRecent
   - Aggregate functions if needed

3. Implement `internal/repository/config_repository.go`
   - Get, Set generic methods
   - GetMaxDailyEffort, SetMaxDailyEffort typed methods

4. Implement `internal/repository/scheduled_task_repository.go`
   - Create, Delete, GetByDate, GetByTask
   - Clear schedule for task
   - Get daily effort total

5. Write repository tests
   - Test CRUD operations
   - Test queries with different conditions
   - Test transaction handling
   - Use in-memory database for tests

**Validation**: All data access works correctly with proper error handling

---

### Phase 4: Scheduler Logic
**Goal**: Implement core scheduling algorithm

#### Tasks:
1. Implement `internal/scheduler/priority.go`
   - `CalculatePriority(task *Task) float64`
   - `SortByPriority(tasks []*Task) []*Task`
   - Test priority calculations

2. Implement `internal/scheduler/scheduler.go`
   - Initialize with repositories
   - `Schedule()` - main scheduling algorithm
   - `ScheduleTask(task *Task)` - schedule single task
   - `GetDailyEffort(date)` - get allocated effort for date
   - `FindNextAvailableDay(effort, startDate)` - find capacity
   - `ClearSchedule(task)` - remove existing schedules

3. Implement scheduling algorithm:
   ```
   a. Get max daily effort
   b. Get all tasks needing scheduling (overdue, unscheduled, or due soon)
   c. Sort by priority
   d. Iterate through next 30 days:
      - For each day, allocate tasks while capacity remains
      - Skip tasks already scheduled
      - Create scheduled_task entries
   ```

4. Write comprehensive scheduler tests
   - Test with various task combinations
   - Test overdue prioritization
   - Test effort limits respected
   - Test rescheduling after completion
   - Test edge cases (no capacity, all tasks scheduled, etc.)

**Validation**: Scheduler produces balanced, valid schedules

---

### Phase 5: CLI Foundation
**Goal**: Create command-line interface using Cobra

#### Tasks:
1. Implement `cmd/chore-scheduler/main.go`
   - Initialize database
   - Setup Cobra root command
   - Handle database path from flag/env
   - Graceful error handling and cleanup

2. Implement `internal/cli/cli.go`
   - CLI struct with dependencies
   - Initialize repositories and scheduler
   - Command registration

3. Create Cobra command structure:
   - Root command with global flags
   - Subcommands for each operation
   - Flag definitions
   - Help text

4. Test CLI initialization
   - Test database connection
   - Test flag parsing

**Validation**: CLI framework runs and parses commands

---

### Phase 6: Task Management Commands
**Goal**: Implement CRUD operations via CLI

#### Tasks:
1. Implement `add` command
   ```bash
   chore-scheduler add "Task name" --effort 2 --frequency 7
   ```
   - Parse arguments and flags
   - Validate input
   - Create task
   - Run scheduler
   - Display confirmation

2. Implement `list` command
   ```bash
   chore-scheduler list [--all | --overdue]
   ```
   - Fetch tasks
   - Format as table with tablewriter
   - Show relevant fields (ID, name, effort, frequency, next due)
   - Color code overdue tasks (optional)

3. Implement `update` command
   ```bash
   chore-scheduler update <id> [--effort N] [--frequency N] [--name "New name"]
   ```
   - Fetch existing task
   - Apply updates
   - Validate
   - Save and reschedule if frequency changed

4. Implement `delete` command
   ```bash
   chore-scheduler delete <id>
   ```
   - Confirm deletion (optional flag to skip)
   - Delete task and cascade

5. Write tests for command handlers
   - Test argument parsing
   - Test validation errors
   - Test success cases

**Validation**: Can manage tasks via CLI

---

### Phase 7: Daily Operations Commands
**Goal**: Implement day-to-day usage commands

#### Tasks:
1. Implement `today` command
   ```bash
   chore-scheduler today
   ```
   - Get tasks scheduled for today
   - Display with effort total
   - Show which are overdue
   - Format nicely

2. Implement `upcoming` command
   ```bash
   chore-scheduler upcoming [--days 7]
   ```
   - Get scheduled tasks for next N days
   - Group by date
   - Show daily effort totals
   - Format as multi-day table

3. Implement `complete` command
   ```bash
   chore-scheduler complete <id>
   ```
   - Validate task exists
   - Create completion record
   - Update last_completed
   - Calculate next_scheduled
   - Clear current schedule
   - Run scheduler
   - Display next due date

4. Implement `postpone` command
   ```bash
   chore-scheduler postpone <id> [--days 1]
   ```
   - Remove from current schedule
   - Optionally adjust next_scheduled
   - Run scheduler to assign new date
   - Display new scheduled date

5. Write tests for daily operations
   - Test completion workflow
   - Test postpone logic
   - Test display formatting

**Validation**: Can complete daily chore workflow via CLI

---

### Phase 8: Configuration and Utilities
**Goal**: Implement configuration management and helper commands

#### Tasks:
1. Implement `config` command
   ```bash
   chore-scheduler config get max-effort
   chore-scheduler config set max-effort 12
   chore-scheduler config list
   ```
   - Get/set config values
   - Validate values
   - Display all config

2. Implement `reschedule` command
   ```bash
   chore-scheduler reschedule [--force]
   ```
   - Clear all schedules
   - Run scheduler from scratch
   - Display summary

3. Implement `stats` command (optional enhancement)
   ```bash
   chore-scheduler stats [--days 30]
   ```
   - Completion rate
   - Tasks completed
   - Average effort per day
   - Streak information

4. Add better error messages and help text
   - Improve all command help
   - Add examples to help text
   - Better error messages

**Validation**: Configuration works and utilities are helpful

---

### Phase 9: Testing and Refinement
**Goal**: Comprehensive testing and bug fixes

#### Tasks:
1. Achieve >80% code coverage
   - Write missing tests
   - Test edge cases
   - Test error paths

2. Integration testing
   - Test complete workflows
   - Test multiple users scenario
   - Test long-term usage patterns

3. Manual testing
   - Run through all commands
   - Test on target NAS environment
   - Test from different devices via SSH

4. Performance testing
   - Test with large number of tasks (100+)
   - Test database performance
   - Optimize queries if needed

5. Documentation
   - Complete README.md
   - Add examples
   - Document installation
   - Create user guide

**Validation**: Application is stable, well-tested, and documented

---

### Phase 10: Build and Deployment
**Goal**: Package for distribution

#### Tasks:
1. Create Makefile
   - build, test, clean targets
   - Cross-compilation for different architectures
   - install target

2. Add version information
   - Build-time version injection
   - `version` command

3. Create releases
   - Build for Linux AMD64 (common NAS)
   - Build for Linux ARM64 (some NAS devices)
   - Create release archives

4. Deployment documentation
   - Installation instructions
   - Systemd service file (optional)
   - Setup guide for NAS

5. Create backup/restore functionality
   - Export database
   - Import from backup

**Validation**: Can build and deploy to NAS

---

## Development Best Practices

### Code Quality
- Follow Go best practices and idioms
- Use meaningful variable names
- Keep functions small and focused
- Add comments for complex logic
- Run `go fmt` and `go vet`

### Testing
- Write tests alongside implementation
- Use table-driven tests where appropriate
- Test happy path and error cases
- Use testify for assertions
- Mock external dependencies when needed

### Error Handling
- Return errors, don't panic
- Wrap errors with context
- Use custom error types for domain errors
- Log errors appropriately
- Provide helpful CLI error messages

### Database
- Use prepared statements
- Handle NULL values properly
- Use transactions for multi-step operations
- Close resources properly (defer)
- Index foreign keys and frequently queried columns

### Git Workflow
- Commit frequently with clear messages
- One feature per commit when possible
- Create branches for larger features
- Tag releases

## Sample Test Cases

### Scheduler Test Case
```go
func TestScheduler_BasicScheduling(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    
    // Setup
    taskRepo := repository.NewTaskRepository(db)
    configRepo := repository.NewConfigRepository(db)
    scheduledRepo := repository.NewScheduledTaskRepository(db)
    
    configRepo.SetMaxDailyEffort(10)
    
    // Create tasks
    tasks := []*models.Task{
        {Name: "Easy task", Effort: 1, FrequencyDays: 2},
        {Name: "Medium task", Effort: 2, FrequencyDays: 3},
        {Name: "Hard task", Effort: 3, FrequencyDays: 5},
    }
    
    for _, task := range tasks {
        err := taskRepo.Create(task)
        require.NoError(t, err)
    }
    
    // Run scheduler
    scheduler := NewScheduler(taskRepo, configRepo, scheduledRepo)
    err := scheduler.Schedule()
    require.NoError(t, err)
    
    // Verify all tasks are scheduled
    for _, task := range tasks {
        scheduled, err := scheduledRepo.GetByTask(task.ID)
        require.NoError(t, err)
        assert.NotNil(t, scheduled)
    }
    
    // Verify no day exceeds max effort
    for i := 0; i < 7; i++ {
        date := time.Now().AddDate(0, 0, i)
        effort, err := scheduler.GetDailyEffort(date)
        require.NoError(t, err)
        assert.LessOrEqual(t, effort, 10)
    }
}
```

## Common Pitfalls to Avoid

1. **Time Zone Issues**: Always use UTC internally, convert for display
2. **NULL Handling**: Use pointers for nullable time fields
3. **SQL Injection**: Use parameterized queries
4. **Resource Leaks**: Always close database connections and statements
5. **Concurrent Access**: SQLite has limited concurrency; use WAL mode
6. **Float Comparison**: Use epsilon for floating point comparisons
7. **Date Arithmetic**: Be careful with date boundaries and edge cases

## Recommended Development Order

1. Start with database and models (foundation)
2. Build repository layer (data access)
3. Implement scheduler (core logic)
4. Add CLI commands incrementally
5. Test continuously
6. Document as you go

This approach allows testing each layer before building on top of it.
