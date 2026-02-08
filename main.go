package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/Utkarsh736/gator/internal/config"
	"github.com/Utkarsh736/gator/internal/database"
	_ "github.com/lib/pq"
)

func main() {
	// Read the config file
	cfg, err := config.Read()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading config: %v\n", err)
		os.Exit(1)
	}

	// Open database connection
	db, err := sql.Open("postgres", cfg.DbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Create database queries
	dbQueries := database.New(db)

	// Initialize application state
	appState := &state{
		db:  dbQueries,
		cfg: &cfg,
	}

	// Initialize commands registry
	cmds := &commands{
		handlers: make(map[string]func(*state, command) error),
	}

	// Register command handlers
	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("reset", handlerReset)
	cmds.register("users", handlerUsers)

	// Parse command-line arguments
	args := os.Args
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Error: not enough arguments provided")
		fmt.Fprintln(os.Stderr, "Usage: gator <command> [args...]")
		os.Exit(1)
	}

	// Create command from args
	cmdName := args[1]
	cmdArgs := []string{}
	if len(args) > 2 {
		cmdArgs = args[2:]
	}

	cmd := command{
		name: cmdName,
		args: cmdArgs,
	}

	// Run the command
	err = cmds.run(appState, cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

