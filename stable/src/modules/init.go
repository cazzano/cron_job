package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func main() {
	fmt.Println("Starting scheduled sync, will run every 60 minutes")
	fmt.Println("Press Ctrl+C to stop the program")

	for {
		// Run the sync operation
		runSync()

		// Print next run time
		nextRun := time.Now().Add(60 * time.Minute)
		fmt.Printf("Next sync will run at: %s\n", nextRun.Format("15:04:05"))

		// Wait for 60 minutes
		time.Sleep(60 * time.Minute)
	}
}

func runSync() {
	fmt.Println("\n------------------------------------------")
	fmt.Println("Starting rclone sync from ~/db to gdrive:Data")
	fmt.Println("Current time:", time.Now().Format("15:04:05"))
	fmt.Println("------------------------------------------")

	// Get the home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		return
	}

	// Create the full path to the db directory in the home folder
	dbPath := filepath.Join(homeDir, "db")

	// Create the command to run rclone sync with the home directory path
	cmd := exec.Command("rclone", "sync", dbPath, "gdrive:Data", "-P")

	// Set the command to use the current terminal
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	fmt.Printf("Executing: rclone sync %s gdrive:Data -P\n", dbPath)
	err = cmd.Run()

	// Check for errors
	if err != nil {
		fmt.Printf("Error executing rclone command: %v\n", err)
	} else {
		fmt.Println("Sync completed successfully")
	}

	fmt.Println("------------------------------------------")
}
