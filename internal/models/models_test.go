package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTask_Validate(t *testing.T) {
	tests := []struct {
		name    string
		task    Task
		wantErr error
	}{
		{
			name:    "valid task",
			task:    Task{Name: "Test", Room: "Kitchen", Effort: 2, FrequencyDays: 7},
			wantErr: nil,
		},
		{
			name:    "empty name",
			task:    Task{Name: "", Room: "Kitchen", Effort: 2, FrequencyDays: 7},
			wantErr: ErrEmptyName,
		},
		{
			name:    "empty room",
			task:    Task{Name: "Test", Room: "", Effort: 2, FrequencyDays: 7},
			wantErr: ErrEmptyRoom,
		},
		{
			name:    "effort too low",
			task:    Task{Name: "Test", Room: "Kitchen", Effort: 0, FrequencyDays: 7},
			wantErr: ErrInvalidEffort,
		},
		{
			name:    "effort too high",
			task:    Task{Name: "Test", Room: "Kitchen", Effort: 4, FrequencyDays: 7},
			wantErr: ErrInvalidEffort,
		},
		{
			name:    "frequency zero",
			task:    Task{Name: "Test", Room: "Kitchen", Effort: 2, FrequencyDays: 0},
			wantErr: ErrInvalidFrequency,
		},
		{
			name:    "frequency negative",
			task:    Task{Name: "Test", Room: "Kitchen", Effort: 2, FrequencyDays: -1},
			wantErr: ErrInvalidFrequency,
		},
		{
			name:    "minimum valid values",
			task:    Task{Name: "T", Room: "R", Effort: 1, FrequencyDays: 1},
			wantErr: nil,
		},
		{
			name:    "maximum effort",
			task:    Task{Name: "Test", Room: "Kitchen", Effort: 3, FrequencyDays: 7},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.task.Validate()
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func TestTask_DaysUntilDue(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		task     Task
		expected int
	}{
		{
			name:     "nil next scheduled",
			task:     Task{NextScheduled: nil},
			expected: -1,
		},
		{
			name:     "due today",
			task:     Task{NextScheduled: &today},
			expected: 0,
		},
		{
			name: "due tomorrow",
			task: Task{NextScheduled: func() *time.Time {
				t := today.AddDate(0, 0, 1)
				return &t
			}()},
			expected: 1,
		},
		{
			name: "due in 7 days",
			task: Task{NextScheduled: func() *time.Time {
				t := today.AddDate(0, 0, 7)
				return &t
			}()},
			expected: 7,
		},
		{
			name: "overdue by 1 day",
			task: Task{NextScheduled: func() *time.Time {
				t := today.AddDate(0, 0, -1)
				return &t
			}()},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.task.DaysUntilDue()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTask_IsOverdue(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		task     Task
		expected bool
	}{
		{
			name:     "nil next scheduled",
			task:     Task{NextScheduled: nil},
			expected: false,
		},
		{
			name:     "due today - not overdue",
			task:     Task{NextScheduled: &today},
			expected: false,
		},
		{
			name: "due tomorrow - not overdue",
			task: Task{NextScheduled: func() *time.Time {
				t := today.AddDate(0, 0, 1)
				return &t
			}()},
			expected: false,
		},
		{
			name: "due yesterday - overdue",
			task: Task{NextScheduled: func() *time.Time {
				t := today.AddDate(0, 0, -1)
				return &t
			}()},
			expected: true,
		},
		{
			name: "due 5 days ago - overdue",
			task: Task{NextScheduled: func() *time.Time {
				t := today.AddDate(0, 0, -5)
				return &t
			}()},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.task.IsOverdue()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTask_DaysOverdue(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		task     Task
		expected int
	}{
		{
			name:     "nil next scheduled",
			task:     Task{NextScheduled: nil},
			expected: 0,
		},
		{
			name:     "due today",
			task:     Task{NextScheduled: &today},
			expected: 0,
		},
		{
			name: "due tomorrow",
			task: Task{NextScheduled: func() *time.Time {
				t := today.AddDate(0, 0, 1)
				return &t
			}()},
			expected: 0,
		},
		{
			name: "due yesterday",
			task: Task{NextScheduled: func() *time.Time {
				t := today.AddDate(0, 0, -1)
				return &t
			}()},
			expected: 1,
		},
		{
			name: "due 5 days ago",
			task: Task{NextScheduled: func() *time.Time {
				t := today.AddDate(0, 0, -5)
				return &t
			}()},
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.task.DaysOverdue()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTask_IsDueToday(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		task     Task
		expected bool
	}{
		{
			name:     "nil next scheduled",
			task:     Task{NextScheduled: nil},
			expected: false,
		},
		{
			name:     "due today",
			task:     Task{NextScheduled: &today},
			expected: true,
		},
		{
			name: "due tomorrow",
			task: Task{NextScheduled: func() *time.Time {
				t := today.AddDate(0, 0, 1)
				return &t
			}()},
			expected: false,
		},
		{
			name: "due yesterday",
			task: Task{NextScheduled: func() *time.Time {
				t := today.AddDate(0, 0, -1)
				return &t
			}()},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.task.IsDueToday()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTask_CalculateNextScheduled(t *testing.T) {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	tests := []struct {
		name          string
		task          Task
		expectedAfter time.Time
	}{
		{
			name:          "no last completed - uses today",
			task:          Task{FrequencyDays: 7, LastCompleted: nil},
			expectedAfter: now.AddDate(0, 0, 6), // at least 6 days from now
		},
		{
			name:          "with last completed",
			task:          Task{FrequencyDays: 7, LastCompleted: &yesterday},
			expectedAfter: yesterday.AddDate(0, 0, 6),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.task.CalculateNextScheduled()
			assert.True(t, result.After(tt.expectedAfter) || result.Equal(tt.expectedAfter.AddDate(0, 0, 1)))
		})
	}
}

func TestCompletion_Validate(t *testing.T) {
	tests := []struct {
		name       string
		completion Completion
		wantErr    error
	}{
		{
			name:       "valid completion",
			completion: Completion{TaskID: 1},
			wantErr:    nil,
		},
		{
			name:       "zero task ID",
			completion: Completion{TaskID: 0},
			wantErr:    ErrInvalidTaskID,
		},
		{
			name:       "negative task ID",
			completion: Completion{TaskID: -1},
			wantErr:    ErrInvalidTaskID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.completion.Validate()
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func TestNewCompletion(t *testing.T) {
	before := time.Now()
	c := NewCompletion(42)
	after := time.Now()

	assert.Equal(t, int64(42), c.TaskID)
	assert.True(t, c.CompletedAt.After(before) || c.CompletedAt.Equal(before))
	assert.True(t, c.CompletedAt.Before(after) || c.CompletedAt.Equal(after))
}

func TestConfig_AsInt(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		want    int
		wantErr bool
	}{
		{
			name:    "valid integer",
			config:  Config{Key: "test", Value: "10"},
			want:    10,
			wantErr: false,
		},
		{
			name:    "zero",
			config:  Config{Key: "test", Value: "0"},
			want:    0,
			wantErr: false,
		},
		{
			name:    "negative",
			config:  Config{Key: "test", Value: "-5"},
			want:    -5,
			wantErr: false,
		},
		{
			name:    "invalid string",
			config:  Config{Key: "test", Value: "abc"},
			want:    0,
			wantErr: true,
		},
		{
			name:    "empty string",
			config:  Config{Key: "test", Value: ""},
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.config.AsInt()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestScheduledTask_NewScheduledTask(t *testing.T) {
	date := time.Date(2024, 1, 15, 14, 30, 0, 0, time.Local)
	st := NewScheduledTask(42, date)

	assert.Equal(t, int64(42), st.TaskID)
	// Should be normalized to midnight UTC
	expected := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, st.ScheduledDate)
}

func TestScheduledTask_IsSameDate(t *testing.T) {
	st := NewScheduledTask(1, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC))

	tests := []struct {
		name     string
		date     time.Time
		expected bool
	}{
		{
			name:     "same date midnight UTC",
			date:     time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "same date different time",
			date:     time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "different date",
			date:     time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC),
			expected: false,
		},
		{
			name:     "day before",
			date:     time.Date(2024, 1, 14, 0, 0, 0, 0, time.UTC),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := st.IsSameDate(tt.date)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTask_CleanlinessStatus(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name          string
		task          Task
		expected      string
	}{
		{
			name:     "never completed - unknown",
			task:     Task{FrequencyDays: 7, LastCompleted: nil},
			expected: CleanlinessUnknown,
		},
		{
			name: "completed today - clean",
			task: Task{FrequencyDays: 7, LastCompleted: func() *time.Time {
				t := now
				return &t
			}()},
			expected: CleanlinessClean,
		},
		{
			name: "completed 3 days ago with 7 day frequency - clean",
			task: Task{FrequencyDays: 7, LastCompleted: func() *time.Time {
				t := now.AddDate(0, 0, -3)
				return &t
			}()},
			expected: CleanlinessClean,
		},
		{
			name: "completed 7 days ago with 7 day frequency - due",
			task: Task{FrequencyDays: 7, LastCompleted: func() *time.Time {
				t := now.AddDate(0, 0, -7)
				return &t
			}()},
			expected: CleanlinessDue,
		},
		{
			name: "completed 10 days ago with 7 day frequency - due (1.4x)",
			task: Task{FrequencyDays: 7, LastCompleted: func() *time.Time {
				t := now.AddDate(0, 0, -10)
				return &t
			}()},
			expected: CleanlinessDue,
		},
		{
			name: "completed 11 days ago with 7 day frequency - dirty (1.57x)",
			task: Task{FrequencyDays: 7, LastCompleted: func() *time.Time {
				t := now.AddDate(0, 0, -11)
				return &t
			}()},
			expected: CleanlinessDirty,
		},
		{
			name: "completed 14 days ago with 7 day frequency - very dirty (2x)",
			task: Task{FrequencyDays: 7, LastCompleted: func() *time.Time {
				t := now.AddDate(0, 0, -14)
				return &t
			}()},
			expected: CleanlinessDirty,
		},
		{
			name: "completed 15 days ago with 7 day frequency - very dirty (2.1x)",
			task: Task{FrequencyDays: 7, LastCompleted: func() *time.Time {
				t := now.AddDate(0, 0, -15)
				return &t
			}()},
			expected: CleanlinessVeryDirty,
		},
		{
			name: "completed 30 days ago with 7 day frequency - very dirty",
			task: Task{FrequencyDays: 7, LastCompleted: func() *time.Time {
				t := now.AddDate(0, 0, -30)
				return &t
			}()},
			expected: CleanlinessVeryDirty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.task.CleanlinessStatus()
			assert.Equal(t, tt.expected, result)
		})
	}
}
