package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

// PIDInfo stores the process ID information
type PIDInfo struct {
	PID       int       `json:"pid"`
	StartTime time.Time `json:"start_time"`
}

func main() {
	// Get the home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		return
	}

	// Construct the path to the PID file
	pidFile := filepath.Join(homeDir, ".config", "cron", "pid.json")

	// Check if the PID file exists
	if _, err := os.Stat(pidFile); os.IsNotExist(err) {
		fmt.Println("No background service seems to be running (PID file not found)")
		return
	}

	// Read the PID file
	data, err := os.ReadFile(pidFile)
	if err != nil {
		fmt.Printf("Error reading PID file: %v\n", err)
		return
	}

	// Parse the JSON data
	var pidInfo PIDInfo
	err = json.Unmarshal(data, &pidInfo)
	if err != nil {
		fmt.Printf("Error parsing PID file: %v\n", err)
		return
	}

	// Display information about the running process
	fmt.Printf("Found process with PID: %d\n", pidInfo.PID)
	fmt.Printf("Process started at: %s\n", pidInfo.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("Process running for: %s\n", time.Since(pidInfo.StartTime).Round(time.Second))

	// Ask for confirmation
	fmt.Print("Do you want to terminate this process? (y/n): ")
	var response string
	fmt.Scanln(&response)

	if response != "y" && response != "Y" {
		fmt.Println("Operation cancelled")
		return
	}

	// Try to kill the process
	process, err := os.FindProcess(pidInfo.PID)
	if err != nil {
		fmt.Printf("Error finding process: %v\n", err)
		return
	}

	// Send a termination signal to the process
	fmt.Printf("Sending termination signal to process %d...\n", pidInfo.PID)
	err = process.Signal(syscall.SIGTERM)

	if err != nil {
		fmt.Printf("Error sending termination signal: %v\n", err)

		// Ask if we should force kill
		fmt.Print("Terminate forcefully? (y/n): ")
		fmt.Scanln(&response)

		if response == "y" || response == "Y" {
			err = process.Kill()
			if err != nil {
				fmt.Printf("Error forcefully terminating process: %v\n", err)

				// As a last resort, try with the kill command
				fmt.Println("Attempting to terminate with system kill command...")

				killCmd := filepath.Join("/bin", "kill")
				if _, err := os.Stat(killCmd); os.IsNotExist(err) {
					killCmd = "kill"
				}

				pid := strconv.Itoa(pidInfo.PID)
				_, err = exec.Command(killCmd, "-9", pid).Output()
				if err != nil {
					fmt.Printf("Failed to force kill: %v\n", err)
					fmt.Println("You may need to manually kill the process")
					return
				}

				fmt.Printf("Process %d has been forcefully terminated\n", pidInfo.PID)
			} else {
				fmt.Printf("Process %d has been forcefully terminated\n", pidInfo.PID)
			}
		} else {
			fmt.Println("Process was not terminated")
			return
		}
	} else {
		fmt.Printf("Termination signal sent to process %d\n", pidInfo.PID)

		// Wait a moment to see if the process exits gracefully
		time.Sleep(2 * time.Second)

		// Check if the process still exists
		err = process.Signal(syscall.Signal(0))
		if err == nil {
			fmt.Println("Process is still running. It may take a moment to shut down properly.")
		} else {
			fmt.Println("Process has been terminated successfully")
		}
	}

	// Since our daemon should clean up its own PID file, we don't need to remove it here
	// If it doesn't get removed, it suggests the daemon didn't shut down properly

	// Check if the PID file still exists after termination attempt
	time.Sleep(1 * time.Second)
	if _, err := os.Stat(pidFile); err == nil {
		fmt.Println("Note: PID file still exists. This might indicate the process didn't shut down properly.")
		fmt.Print("Remove the PID file? (y/n): ")
		fmt.Scanln(&response)

		if response == "y" || response == "Y" {
			err = os.Remove(pidFile)
			if err != nil {
				fmt.Printf("Error removing PID file: %v\n", err)
			} else {
				fmt.Println("PID file removed successfully")
			}
		}
	}
}
