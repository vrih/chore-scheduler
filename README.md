# Chore Scheduler

A command-line application for intelligent scheduling of recurring household tasks based on effort levels and desired frequencies.

## Overview

Chore Scheduler helps you maintain consistent completion of household tasks by:
- Automatically distributing tasks across days based on effort limits
- Prioritizing overdue and urgent tasks
- Maintaining a balanced daily workload
- Tracking completion history

## Features

- **Smart Scheduling**: Automatically schedules tasks to meet frequency requirements while respecting daily effort limits
- **Priority System**: Prioritizes overdue tasks and tasks approaching their due date
- **Simple CLI**: Easy-to-use command-line interface accessible over SSH
- **SQLite Storage**: Lightweight, single-file database with no external dependencies
- **Cross-Platform**: Single binary that runs on Linux, macOS, and Windows
- **Unit Tested**: Comprehensive test coverage for reliability

## Installation

### From Source
```bash
git clone https://github.com/user/chore-scheduler.git
cd chore-scheduler
make build
sudo make install
```

### Pre-built Binaries
Download the latest release for your platform from the [releases page](https://github.com/user/chore-scheduler/releases).

For NAS deployment:
```bash
# Linux AMD64 (most common)
wget https://github.com/user/chore-scheduler/releases/latest/download/chore-scheduler-linux-amd64
chmod +x chore-scheduler-linux-amd64
sudo mv chore-scheduler-linux-amd64 /usr/local/bin/chore-scheduler

# Linux ARM64
wget https://github.com/user/chore-scheduler/releases/latest/download/chore-scheduler-linux-arm64
chmod +x chore-scheduler-linux-arm64
sudo mv chore-scheduler-linux-arm64 /usr/local/bin/chore-scheduler
```

## Quick Start

### 1. Set Maximum Daily Effort
```bash
chore-scheduler config set max-effort 10
```

### 2. Add Some Tasks
```bash
# Format: add "task name" --effort <1-3> --frequency <days>
chore-scheduler add "Vacuum living room" --effort 2 --frequency 3
chore-scheduler add "Clean bathroom" --effort 3 --frequency 7
chore-scheduler add "Water plants" --effort 1 --frequency 2
chore-scheduler add "Wash dishes" --effort 2 --frequency 1
chore-scheduler add "Take out trash" --effort 1 --frequency 3
```

### 3. View Today's Tasks
```bash
chore-scheduler today
```

### 4. Complete a Task
```bash
chore-scheduler complete 1
```

## Usage

### Task Management

#### Add a Task
```bash
chore-scheduler add "Task name" --effort <1-3> --frequency <days>

# Examples:
chore-scheduler add "Mop floors" --effort 3 --frequency 14
chore-scheduler add "Dust shelves" --effort 1 --frequency 7
```

**Effort Levels:**
- `1` - Quick task (5-15 minutes)
- `2` - Medium task (15-45 minutes)
- `3` - Long task (45+ minutes)

**Frequency:** Number of days between completions

#### List All Tasks
```bash
chore-scheduler list

# Show only overdue tasks
chore-scheduler list --overdue
```

#### Update a Task
```bash
chore-scheduler update <id> [--effort <1-3>] [--frequency <days>] [--name "New name"]

# Examples:
chore-scheduler update 1 --frequency 5
chore-scheduler update 2 --effort 2 --frequency 10
chore-scheduler update 3 --name "Deep clean bathroom"
```

#### Delete a Task
```bash
chore-scheduler delete <id>

# Skip confirmation prompt
chore-scheduler delete <id> --force
```

### Daily Operations

#### View Today's Schedule
```bash
chore-scheduler today
```

Output example:
```
Today's Tasks (Effort: 7/10)
+----+------------------+--------+----------+
| ID | Task             | Effort | Status   |
+----+------------------+--------+----------+
|  1 | Vacuum living rm |      2 | Due      |
|  3 | Water plants     |      1 | Due      |
|  5 | Take out trash   |      1 | Due      |
|  2 | Clean bathroom   |      3 | OVERDUE! |
+----+------------------+--------+----------+
```

#### View Upcoming Week
```bash
chore-scheduler upcoming

# Show more days
chore-scheduler upcoming --days 14
```

#### Complete a Task
```bash
chore-scheduler complete <id>
```

This will:
- Record the completion
- Calculate the next due date
- Reschedule the task
- Display when the task is next due

#### Postpone a Task
```bash
chore-scheduler postpone <id>

# Postpone by specific number of days
chore-scheduler postpone <id> --days 2
```

### Configuration

#### View Configuration
```bash
chore-scheduler config list
chore-scheduler config get max-effort
```

#### Set Maximum Daily Effort
```bash
chore-scheduler config set max-effort 12
```

#### Force Reschedule All Tasks
```bash
chore-scheduler reschedule
```

Use this after:
- Changing max daily effort
- Completing multiple tasks
- Making significant changes to task frequencies

### Advanced Usage

#### Custom Database Location
```bash
# Via flag
chore-scheduler --db /path/to/database.db today

# Via environment variable
export CHORE_SCHEDULER_DB=/nas/shared/chores.db
chore-scheduler today
```

#### View Statistics (if implemented)
```bash
chore-scheduler stats
chore-scheduler stats --days 30
```

## Architecture

### Technology Stack
- **Language**: Go 1.21+
- **Database**: SQLite 3
- **CLI Framework**: Cobra
- **Testing**: Go testing + testify

### Project Structure
```
chore-scheduler/
├── cmd/chore-scheduler/    # Application entry point
├── internal/
│   ├── db/                 # Database layer
│   ├── models/             # Data models
│   ├── repository/         # Data access layer
│   ├── scheduler/          # Scheduling logic
│   └── cli/                # CLI commands
├── go.mod
└── Makefile
```

### Database Schema
- **tasks**: Task definitions and scheduling metadata
- **completions**: Audit log of task completions
- **scheduled_tasks**: Daily task assignments
- **config**: Application configuration

## Development

### Build
```bash
make build
```

### Test
```bash
make test

# With coverage
go test -v -cover ./...
```

### Run
```bash
make run

# Or directly
go run cmd/chore-scheduler/main.go
```

### Cross-Compile
```bash
# For Linux AMD64
make build-linux-amd64

# For Linux ARM64
make build-linux-arm64
```

## How It Works

### Scheduling Algorithm

1. **Priority Calculation**: Tasks are prioritized by:
   - Overdue tasks get highest priority (1000 + days overdue)
   - Tasks due soon get priority based on urgency (100 / days until due)
   - Unscheduled tasks get medium priority (50)

2. **Daily Assignment**: For each day starting from today:
   - Calculate remaining effort capacity (max effort - already scheduled)
   - Assign highest priority tasks that fit within capacity
   - Continue until capacity is full or all tasks are scheduled

3. **Rescheduling Triggers**:
   - When a task is completed (reschedule for next due date)
   - When a task is postponed (find next available day)
   - When configuration changes (max effort adjusted)
   - Manual reschedule command

### Task Lifecycle

```
Add Task → Initial Schedule → [Due → Complete → Reschedule] → (repeat)
                           ↓
                        Postpone → Reschedule
```

## Configuration File Location

Default database location:
- Linux/macOS: `~/.chore-scheduler/chore.db`
- Windows: `%USERPROFILE%\.chore-scheduler\chore.db`

Override with:
- `--db` flag: `chore-scheduler --db /path/to/db.sqlite`
- Environment variable: `CHORE_SCHEDULER_DB=/path/to/db.sqlite`

## Tips for Use

1. **Start Small**: Begin with 5-10 tasks and adjust effort levels as you learn
2. **Realistic Effort**: Set max daily effort based on actual available time
3. **Adjust Frequencies**: Fine-tune task frequencies after a week or two
4. **Regular Reviews**: Run `chore-scheduler list` weekly to review all tasks
5. **Postpone Wisely**: Use postpone for special circumstances, not as a habit
6. **Completion Habit**: Mark tasks complete immediately after finishing

## Troubleshooting

### Database locked error
SQLite allows limited concurrent access. If running multiple instances:
```bash
# Enable WAL mode in database (better concurrency)
sqlite3 ~/.chore-scheduler/chore.db "PRAGMA journal_mode=WAL;"
```

### Tasks not showing up today
```bash
# Force reschedule
chore-scheduler reschedule

# Check if max effort is too low
chore-scheduler config get max-effort
```

### Effort always at limit
Increase max daily effort or reduce task frequencies:
```bash
chore-scheduler config set max-effort 15
```

## Future Enhancements (Phase 2)

- Web interface for mobile-friendly access
- JSON API for app integration
- Task categories and tags
- Multiple users/households
- Statistics dashboard
- Completion streaks
- Task dependencies
- Recurring patterns (e.g., "every Monday")
- Mobile notifications

## Contributing

Contributions are welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

MIT License - see LICENSE file for details

## Support

For issues, questions, or suggestions:
- Open an issue on GitHub
- Check existing issues for solutions
- Read the documentation in `/docs`

## Acknowledgments

Built to replace Sweepy with a lightweight, self-hosted alternative optimized for SSH access and terminal usage.
