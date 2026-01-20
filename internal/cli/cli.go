package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/user/chore-scheduler/internal/db"
	"github.com/user/chore-scheduler/internal/repository"
	"github.com/user/chore-scheduler/internal/scheduler"
)

// CLI holds all dependencies needed for command execution
type CLI struct {
	db             *db.DB
	taskRepo       repository.TaskRepository
	completionRepo repository.CompletionRepository
	configRepo     repository.ConfigRepository
	scheduledRepo  repository.ScheduledTaskRepository
	scheduler      *scheduler.Scheduler
	rootCmd        *cobra.Command
	dbPath         string
}

// New creates a new CLI instance
func New() *CLI {
	cli := &CLI{}
	cli.rootCmd = cli.buildRootCommand()
	return cli
}

// Execute runs the CLI application
func (c *CLI) Execute() error {
	return c.rootCmd.Execute()
}

// buildRootCommand creates the root command with all subcommands
func (c *CLI) buildRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "chore-scheduler",
		Short: "Intelligent task scheduling for household chores",
		Long: `Chore Scheduler is a CLI tool for managing and scheduling household chores.
It intelligently distributes tasks across days based on effort levels and frequency.`,
		PersistentPreRunE: c.initializeDatabase,
		PersistentPostRun: c.cleanup,
		SilenceUsage:      true,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&c.dbPath, "db", "", "database file path (default: ~/.chore-scheduler/chore.db)")

	// Add subcommands
	rootCmd.AddCommand(c.buildAddCommand())
	rootCmd.AddCommand(c.buildListCommand())
	rootCmd.AddCommand(c.buildUpdateCommand())
	rootCmd.AddCommand(c.buildDeleteCommand())
	rootCmd.AddCommand(c.buildTodayCommand())
	rootCmd.AddCommand(c.buildUpcomingCommand())
	rootCmd.AddCommand(c.buildCompleteCommand())
	rootCmd.AddCommand(c.buildPostponeCommand())
	rootCmd.AddCommand(c.buildConfigCommand())
	rootCmd.AddCommand(c.buildRescheduleCommand())
	rootCmd.AddCommand(c.buildRoomsCommand())

	return rootCmd
}

// initializeDatabase sets up the database connection and repositories
func (c *CLI) initializeDatabase(cmd *cobra.Command, args []string) error {
	dbPath := c.dbPath
	if dbPath == "" {
		// Check environment variable
		dbPath = os.Getenv("CHORE_SCHEDULER_DB")
	}
	if dbPath == "" {
		// Use default path
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		dbPath = filepath.Join(homeDir, ".chore-scheduler", "chore.db")
	}

	// Create database connection
	database, err := db.New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	c.db = database

	// Initialize schema
	if err := database.Initialize(); err != nil {
		database.Close()
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Create repositories
	c.taskRepo = repository.NewTaskRepository(database)
	c.completionRepo = repository.NewCompletionRepository(database)
	c.configRepo = repository.NewConfigRepository(database)
	c.scheduledRepo = repository.NewScheduledTaskRepository(database)

	// Create scheduler
	c.scheduler = scheduler.NewScheduler(c.taskRepo, c.configRepo, c.scheduledRepo)

	return nil
}

// cleanup closes the database connection
func (c *CLI) cleanup(cmd *cobra.Command, args []string) {
	if c.db != nil {
		c.db.Close()
	}
}

// getDefaultDBPath returns the default database path
func getDefaultDBPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "chore.db"
	}
	return filepath.Join(homeDir, ".chore-scheduler", "chore.db")
}
