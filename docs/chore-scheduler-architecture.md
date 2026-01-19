# Chore Scheduler - Technical Architecture

## Project Structure

```
chore-scheduler/
├── cmd/
│   └── chore-scheduler/
│       └── main.go                 # Application entry point
├── internal/
│   ├── db/
│   │   ├── db.go                   # Database connection and initialization
│   │   ├── migrations.go           # Schema migrations
│   │   └── db_test.go
│   ├── models/
│   │   ├── task.go                 # Task model and methods
│   │   ├── completion.go           # Completion model
│   │   ├── config.go               # Configuration model
│   │   └── models_test.go
│   ├── scheduler/
│   │   ├── scheduler.go            # Core scheduling logic
│   │   ├── priority.go             # Priority calculation
│   │   └── scheduler_test.go
│   ├── repository/
│   │   ├── task_repository.go      # Task data access
│   │   ├── completion_repository.go
│   │   ├── config_repository.go
│   │   └── repository_test.go
│   └── cli/
│       ├── commands.go             # CLI command definitions
│       ├── handlers.go             # Command handlers
│       └── cli_test.go
├── go.mod
├── go.sum
├── README.md
└── Makefile
```

## Database Schema

### Tasks Table
```sql
CREATE TABLE tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    effort INTEGER NOT NULL CHECK(effort >= 1 AND effort <= 3),
    frequency_days INTEGER NOT NULL CHECK(frequency_days > 0),
    last_completed DATETIME,
    next_scheduled DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tasks_next_scheduled ON tasks(next_scheduled);
CREATE INDEX idx_tasks_last_completed ON tasks(last_completed);
```

### Completions Table
```sql
CREATE TABLE completions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    completed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

CREATE INDEX idx_completions_task_id ON completions(task_id);
CREATE INDEX idx_completions_completed_at ON completions(completed_at);
```

### Config Table
```sql
CREATE TABLE config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- Default configuration
INSERT INTO config (key, value) VALUES ('max_daily_effort', '10');
```

### Scheduled Tasks Table
```sql
CREATE TABLE scheduled_tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    scheduled_date DATE NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    UNIQUE(task_id, scheduled_date)
);

CREATE INDEX idx_scheduled_tasks_date ON scheduled_tasks(scheduled_date);
CREATE INDEX idx_scheduled_tasks_task_id ON scheduled_tasks(task_id);
```

## Core Components

### 1. Database Layer (`internal/db`)

**db.go:**
```go
type DB struct {
    *sql.DB
}

func New(dbPath string) (*DB, error)
func (db *DB) Initialize() error
func (db *DB) Close() error
```

**migrations.go:**
```go
func runMigrations(db *sql.DB) error
```

### 2. Models (`internal/models`)

**task.go:**
```go
type Task struct {
    ID            int64
    Name          string
    Effort        int  // 1-3
    FrequencyDays int
    LastCompleted *time.Time
    NextScheduled *time.Time
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

func (t *Task) DaysUntilDue() int
func (t *Task) IsOverdue() bool
func (t *Task) DaysOverdue() int
```

**completion.go:**
```go
type Completion struct {
    ID          int64
    TaskID      int64
    CompletedAt time.Time
}
```

**config.go:**
```go
type Config struct {
    Key   string
    Value string
}
```

### 3. Repository Layer (`internal/repository`)

**task_repository.go:**
```go
type TaskRepository interface {
    Create(task *Task) error
    Get(id int64) (*Task, error)
    GetAll() ([]*Task, error)
    Update(task *Task) error
    Delete(id int64) error
    GetByScheduledDate(date time.Time) ([]*Task, error)
    GetUnscheduled() ([]*Task, error)
    GetOverdue() ([]*Task, error)
}

type taskRepository struct {
    db *DB
}

func NewTaskRepository(db *DB) TaskRepository
```

**completion_repository.go:**
```go
type CompletionRepository interface {
    Create(completion *Completion) error
    GetByTaskID(taskID int64) ([]*Completion, error)
    GetRecent(limit int) ([]*Completion, error)
}

func NewCompletionRepository(db *DB) CompletionRepository
```

**config_repository.go:**
```go
type ConfigRepository interface {
    Get(key string) (string, error)
    Set(key, value string) error
    GetMaxDailyEffort() (int, error)
    SetMaxDailyEffort(effort int) error
}

func NewConfigRepository(db *DB) ConfigRepository
```

### 4. Scheduler (`internal/scheduler`)

**scheduler.go:**
```go
type Scheduler struct {
    taskRepo       TaskRepository
    configRepo     ConfigRepository
    scheduledRepo  ScheduledTaskRepository
}

func NewScheduler(taskRepo, configRepo, scheduledRepo) *Scheduler

// Main scheduling logic
func (s *Scheduler) Schedule() error

// Schedule a specific task to next available day
func (s *Scheduler) ScheduleTask(task *Task) error

// Get effort allocated for a specific date
func (s *Scheduler) GetDailyEffort(date time.Time) (int, error)

// Find next available day with capacity for effort
func (s *Scheduler) FindNextAvailableDay(effort int, startDate time.Time) (time.Time, error)
```

**priority.go:**
```go
// Calculate priority score for task (higher = more urgent)
func CalculatePriority(task *Task) float64

// Sort tasks by priority
func SortByPriority(tasks []*Task) []*Task
```

### 5. CLI Layer (`internal/cli`)

**commands.go:**
```go
type CLI struct {
    taskRepo       TaskRepository
    completionRepo CompletionRepository
    configRepo     ConfigRepository
    scheduler      *Scheduler
}

func NewCLI(db *DB) *CLI

func (c *CLI) Execute(args []string) error
```

**Command Handlers:**
```go
func (c *CLI) handleAdd(args []string) error
func (c *CLI) handleList(args []string) error
func (c *CLI) handleUpdate(args []string) error
func (c *CLI) handleDelete(args []string) error
func (c *CLI) handleToday(args []string) error
func (c *CLI) handleUpcoming(args []string) error
func (c *CLI) handleComplete(args []string) error
func (c *CLI) handlePostpone(args []string) error
func (c *CLI) handleConfig(args []string) error
func (c *CLI) handleReschedule(args []string) error
```

## Scheduling Algorithm Details

### Priority Calculation
```
For each task:
  if task.IsOverdue():
    priority = 1000 + task.DaysOverdue()
  else if task.NextScheduled != nil:
    daysUntil = task.DaysUntilDue()
    if daysUntil <= 0:
      priority = 1000
    else:
      priority = 100.0 / daysUntil
  else:
    priority = 50  // Unscheduled tasks
```

### Scheduling Process
```
1. Get max daily effort from config
2. Get all tasks that need scheduling
3. Sort tasks by priority (descending)
4. For each day starting from today:
   a. Get current effort for day
   b. Get remaining capacity (max - current)
   c. While capacity > 0 and tasks remain:
      - Take highest priority task that fits
      - Assign to current day
      - Reduce capacity
      - Mark task as scheduled
5. Save all schedule changes
```

### On Task Completion
```
1. Record completion in completions table
2. Update task.LastCompleted = now
3. Calculate task.NextScheduled = now + frequency_days
4. Clear any existing schedule entries for this task
5. Run scheduler to assign to new date
```

### On Task Postpone
```
1. Clear scheduled date for task
2. Increase priority temporarily (mark as urgent)
3. Run scheduler to find next available day
```

## CLI Command Examples

```bash
# Add tasks
chore-scheduler add "Vacuum living room" --effort 2 --frequency 3
chore-scheduler add "Clean bathroom" --effort 3 --frequency 7
chore-scheduler add "Water plants" --effort 1 --frequency 2

# List all tasks
chore-scheduler list

# Update task
chore-scheduler update 1 --frequency 5 --effort 2

# Delete task
chore-scheduler delete 1

# View today's tasks
chore-scheduler today

# View upcoming week
chore-scheduler upcoming

# Complete a task
chore-scheduler complete 2

# Postpone a task
chore-scheduler postpone 3

# Configure max effort
chore-scheduler config set max-effort 12
chore-scheduler config get max-effort

# Force reschedule all tasks
chore-scheduler reschedule
```

## Dependencies

```go
// go.mod
module github.com/user/chore-scheduler

go 1.21

require (
    github.com/mattn/go-sqlite3 v1.14.18
    github.com/spf13/cobra v1.8.0
    github.com/stretchr/testify v1.8.4
)
```

### Recommended Libraries
- **CLI Framework**: `github.com/spf13/cobra` - robust CLI with subcommands
- **SQLite Driver**: `github.com/mattn/go-sqlite3` - CGO-based SQLite driver
- **Testing**: `github.com/stretchr/testify` - assertion and mocking
- **Table Output**: `github.com/olekukonko/tablewriter` - formatted table output

## Build and Deployment

### Makefile
```makefile
.PHONY: build test clean run install

build:
	go build -o bin/chore-scheduler cmd/chore-scheduler/main.go

test:
	go test -v -cover ./...

clean:
	rm -rf bin/

run:
	go run cmd/chore-scheduler/main.go

install:
	go install cmd/chore-scheduler/main.go

# Cross-compilation for NAS (common architectures)
build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -o bin/chore-scheduler-linux-amd64 cmd/chore-scheduler/main.go

build-linux-arm64:
	GOOS=linux GOARCH=arm64 go build -o bin/chore-scheduler-linux-arm64 cmd/chore-scheduler/main.go
```

## Configuration

### Database Location
- Default: `~/.chore-scheduler/chore.db`
- Override with `--db` flag or `CHORE_SCHEDULER_DB` environment variable

### Example Usage
```bash
# Use custom database location
chore-scheduler --db /path/to/db.sqlite today

# Or set environment variable
export CHORE_SCHEDULER_DB=/nas/shared/chores.db
chore-scheduler today
```

## Testing Strategy

### Unit Tests
- Test all repository methods with in-memory SQLite
- Test scheduler algorithm with known scenarios
- Test priority calculations
- Test date arithmetic

### Integration Tests
- Test complete workflows (add task → schedule → complete)
- Test CLI command parsing and execution

### Test Database
```go
func setupTestDB(t *testing.T) *DB {
    db, err := New(":memory:")
    require.NoError(t, err)
    require.NoError(t, db.Initialize())
    return db
}
```

## Error Handling

- Use custom error types for domain errors
- Wrap database errors with context
- Provide helpful CLI error messages
- Exit codes: 0 = success, 1 = error

```go
var (
    ErrTaskNotFound = errors.New("task not found")
    ErrInvalidEffort = errors.New("effort must be between 1 and 3")
    ErrInvalidFrequency = errors.New("frequency must be greater than 0")
)
```
