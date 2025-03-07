package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

// PIDInfo stores the process ID information
type PIDInfo struct {
	PID       int       `json:"pid"`
	StartTime time.Time `json:"start_time"`
}

func main() {
	// Check if we're already running as a daemon
	isDaemon := false
	for _, arg := range os.Args {
		if arg == "--daemon" {
			isDaemon = true
			break
		}
	}

	if !isDaemon {
		// We're in the parent process, spawn the daemon and exit
		fmt.Println("Starting scheduled sync, will run every 60 minutes")

		// Get the current executable path
		executable, err := os.Executable()
		if err != nil {
			fmt.Printf("Error getting executable path: %v\n", err)
			return
		}

		// Create the daemon process
		cmd := exec.Command(executable, "--daemon")
		cmd.Stdout = nil // Redirect stdout to null
		cmd.Stderr = nil // Redirect stderr to null

		// Start the daemon process
		err = cmd.Start()
		if err != nil {
			fmt.Printf("Error starting daemon process: %v\n", err)
			return
		}

		// Show the message
		fmt.Printf("Process ID: %d\n", cmd.Process.Pid)
		fmt.Println("\n--- Program is continuing in background services ---")

		// Exit the parent process
		return
	}

	// We're in the daemon process now

	// Save PID information
	savePID()

	// Log that we've started as a daemon
	logMessage("Background service started")

	// Create a done channel for cleanup on termination
	done := make(chan bool, 1)

	// Set up the daemon to handle termination signals
	setupSignalHandling(done)

	// Start the main sync loop
	for {
		// Run the sync operation
		runSync()

		// Log next sync time
		nextRun := time.Now().Add(60 * time.Minute)
		logMessage(fmt.Sprintf("Next sync scheduled for: %s", nextRun.Format("15:04:05")))

		// Wait for the next scheduled time
		select {
		case <-time.After(60 * time.Minute):
			// Time to run the next sync
			continue
		case <-done:
			// Received signal to terminate
			logMessage("Background service shutting down")
			removePIDFile()
			return
		}
	}
}

func setupSignalHandling(done chan bool) {
	// Create a channel to catch signals
	sigs := make(chan os.Signal, 1)
	go func() {
		<-sigs
		logMessage("Termination signal received")
		done <- true
	}()
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
}

func savePID() {
	// Get the home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logMessage(fmt.Sprintf("Error getting home directory: %v", err))
		return
	}

	// Create the config directory if it doesn't exist
	configDir := filepath.Join(homeDir, ".config", "cron")
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		logMessage(fmt.Sprintf("Error creating config directory: %v", err))
		return
	}

	// Create the PID info
	pidInfo := PIDInfo{
		PID:       os.Getpid(),
		StartTime: time.Now(),
	}

	// Marshal to JSON
	pidData, err := json.MarshalIndent(pidInfo, "", "  ")
	if err != nil {
		logMessage(fmt.Sprintf("Error marshaling PID data: %v", err))
		return
	}

	// Write to file
	pidFile := filepath.Join(configDir, "pid.json")
	err = os.WriteFile(pidFile, pidData, 0644)
	if err != nil {
		logMessage(fmt.Sprintf("Error writing PID file: %v", err))
		return
	}

	logMessage(fmt.Sprintf("PID information saved to %s", pidFile))
}

func removePIDFile() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logMessage(fmt.Sprintf("Error getting home directory: %v", err))
		return
	}

	pidFile := filepath.Join(homeDir, ".config", "cron", "pid.json")
	err = os.Remove(pidFile)
	if err != nil {
		logMessage(fmt.Sprintf("Error removing PID file: %v", err))
		return
	}

	logMessage(fmt.Sprintf("PID file removed: %s", pidFile))
}

func runSync() {
	logMessage("Starting rclone sync from ~/db to gdrive:Data")

	// Get the home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logMessage(fmt.Sprintf("Error getting home directory: %v", err))
		return
	}

	// Create the full path to the db directory in the home folder
	dbPath := filepath.Join(homeDir, "db")

	// Create the command to run rclone sync with the home directory path
	cmd := exec.Command("rclone", "sync", dbPath, "gdrive:Data")

	// Run the command
	output, err := cmd.CombinedOutput()

	// Check for errors
	if err != nil {
		logMessage(fmt.Sprintf("Error executing rclone command: %v", err))
		logMessage(string(output))
	} else {
		logMessage("Sync completed successfully")
	}
}

func logMessage(message string) {
	// Get the home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}

	// Create the log directory if it doesn't exist
	logDir := filepath.Join(homeDir, ".config", "cron", "logs")
	err = os.MkdirAll(logDir, 0755)
	if err != nil {
		return
	}

	// Create log file path
	logFile := filepath.Join(logDir, "background_service.log")

	// Open the log file (append or create)
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()

	// Write log entry with timestamp
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s\n", timestamp, message)

	file.WriteString(logEntry)
}
