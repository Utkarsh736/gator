package main

import (
	"fmt"
	"log"

	"github.com/Utkarsh736/gator/internal/config"
)

func main() {
	// Step 1: Read the initial config
	cfg, err := config.Read()
	if err != nil {
		log.Fatalf("Error reading config: %v", err)
	}
	fmt.Println("Initial config:")
	fmt.Printf("%+v\n\n", cfg)

	// Step 2: Set the current user (replace "lane" with your name)
	err = cfg.SetUser("Utkarsh")
	if err != nil {
		log.Fatalf("Error setting user: %v", err)
	}
	fmt.Println("Updated user to: Utkarsh")

	// Step 3: Read the config again to verify
	cfg, err = config.Read()
	if err != nil {
		log.Fatalf("Error reading config after update: %v", err)
	}
	fmt.Println("\nConfig after update:")
	fmt.Printf("%+v\n", cfg)
}

