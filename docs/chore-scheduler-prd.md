# Chore Scheduler - Product Requirements Document

## Overview
A command-line application to replace Sweepy, providing intelligent task scheduling based on effort levels and desired frequencies.

## Core Objectives
- Help users maintain consistent completion of recurring household tasks
- Distribute workload evenly across days based on user-defined effort limits
- Provide simple CLI interface accessible over SSH from multiple devices
- Maintain persistent state in SQLite database

## User Stories

### Task Management
- As a user, I want to add new tasks with effort levels (1-3) and desired frequency
- As a user, I want to delete tasks I no longer need to track
- As a user, I want to modify the frequency of existing tasks
- As a user, I want to postpone a task when I can't complete it as scheduled

### Scheduling
- As a user, I want to set a maximum daily effort limit
- As a user, I want the system to automatically schedule tasks to meet their frequencies
- As a user, I want to see which tasks are scheduled for today
- As a user, I want the scheduler to prioritize overdue or nearly-due tasks

### Task Completion
- As a user, I want to mark tasks as completed
- As a user, I want the system to automatically reschedule completed tasks based on their frequency

## Technical Requirements

### Technology Stack
- Language: Go (Golang)
- Database: SQLite
- Deployment: Single binary executable
- Testing: Unit tests for all core functionality

### Deployment Environment
- Run on NAS via SSH
- Access from phone, tablet, and desktop terminals
- No web interface required (Phase 1)

### Data Model

#### Tasks Table
- `id` (INTEGER PRIMARY KEY)
- `name` (TEXT NOT NULL)
- `effort` (INTEGER 1-3 NOT NULL)
- `frequency_days` (INTEGER NOT NULL) - how often task should be done
- `last_completed` (DATETIME)
- `next_scheduled` (DATETIME)
- `created_at` (DATETIME)
- `updated_at` (DATETIME)

#### Settings Table
- `key` (TEXT PRIMARY KEY)
- `value` (TEXT)
- Store: `max_daily_effort`

#### Completions Table (audit log)
- `id` (INTEGER PRIMARY KEY)
- `task_id` (INTEGER FOREIGN KEY)
- `completed_at` (DATETIME)

### Core Features

#### 1. Task CRUD Operations
```
chore-scheduler add "Clean bathroom" --effort 3 --frequency 7
chore-scheduler list
chore-scheduler update <id> --frequency 14
chore-scheduler delete <id>
```

#### 2. Scheduling Algorithm
- Calculate next due date based on frequency and last completion
- Assign tasks to days while respecting max daily effort
- Prioritize tasks by:
  1. Overdue tasks (past due date)
  2. Tasks due soon
  3. Tasks not yet scheduled
- Rebalance schedule when tasks are completed or postponed

#### 3. Daily Operations
```
chore-scheduler today              # Show today's tasks
chore-scheduler complete <id>      # Mark task complete
chore-scheduler postpone <id>      # Postpone task to next available day
chore-scheduler upcoming           # Show next 7 days schedule
```

#### 4. Configuration
```
chore-scheduler config set max-effort 10
chore-scheduler config get max-effort
```

### Scheduling Logic Details

**Initial Scheduling:**
- When a task is added, calculate `next_scheduled` = current_date + frequency_days
- Run scheduling algorithm to assign to specific day

**Scheduling Algorithm:**
- For each day starting from today:
  - Get unscheduled tasks ordered by priority
  - Add tasks while daily effort < max_effort
  - Store scheduled date for task
  
**Priority Calculation:**
- Overdue tasks: days_overdue (higher = more urgent)
- Not overdue: 1 / days_until_due (higher = more urgent)

**On Completion:**
- Update `last_completed` to now
- Calculate `next_scheduled` = last_completed + frequency_days
- Run rescheduling algorithm for future tasks

**On Postpone:**
- Clear current day assignment
- Mark as high priority for next available day
- Run rescheduling algorithm

### CLI Command Structure

```
chore-scheduler [command] [arguments] [flags]

Commands:
  add         Add a new task
  list        List all tasks
  update      Update task properties
  delete      Delete a task
  today       Show today's scheduled tasks
  upcoming    Show upcoming week schedule
  complete    Mark task as complete
  postpone    Postpone a task
  config      Manage configuration
  reschedule  Force reschedule of all tasks
```

### Testing Requirements
- Unit tests for:
  - Database operations (CRUD)
  - Scheduling algorithm
  - Priority calculations
  - Date calculations
  - Configuration management
- Minimum 80% code coverage
- Tests should use in-memory SQLite database

## Future Considerations (Phase 2)
- Web interface for mobile-friendly access
- JSON API endpoints
- Multiple users/households
- Task categories/tags
- Statistics and completion history
- Recurring task patterns (e.g., "every Monday")
- Task dependencies

## Success Criteria
- Users can manage tasks via CLI commands
- System maintains balanced daily schedule
- All tasks complete within their frequency requirements
- Application runs reliably on NAS
- All core functionality covered by unit tests
