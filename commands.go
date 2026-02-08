package main

import (
	"errors"
	"fmt"

	"github.com/Utkarsh736/gator/internal/config"
)

// state holds the application state (config, later DB connection)
type state struct {
	cfg *config.Config
}

// command represents a CLI command with its name and arguments
type command struct {
	name string
	args []string
}

// commands holds all registered command handlers
type commands struct {
	handlers map[string]func(*state, command) error
}

// register adds a new command handler
func (c *commands) register(name string, f func(*state, command) error) {
	c.handlers[name] = f
}

// run executes a command by name if it exists
func (c *commands) run(s *state, cmd command) error {
	handler, exists := c.handlers[cmd.name]
	if !exists {
		return fmt.Errorf("unknown command: %s", cmd.name)
	}
	return handler(s, cmd)
}

// handlerLogin sets the current user in the config
func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return errors.New("login command requires a username argument")
	}

	username := cmd.args[0]

	err := s.cfg.SetUser(username)
	if err != nil {
		return fmt.Errorf("couldn't set current user: %w", err)
	}

	fmt.Printf("User has been set to: %s\n", username)
	return nil
}

