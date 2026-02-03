package orchestrate

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
)

// killProcByPort uses 'ss' to find PIDs and kills them
func killProcByPort(port int) error {
	// We use 'ss' (socket statistics) which is standard on modern Linux
	// -l: listening sockets
	// -p: show process using socket
	// -t: tcp only
	// -n: numeric ports (don't resolve service names)
	cmd := exec.Command("ss", "-lptn", fmt.Sprintf("sport = :%d", port))
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("error running 'ss': %v (is iproute2 installed?)", err)
	}

	// Regex to find "pid=12345" inside the output
	// Output looks like: users:(("alertmanager",pid=1234,fd=3))
	re := regexp.MustCompile(`pid=(\d+)`)
	matches := re.FindAllStringSubmatch(string(output), -1)

	if len(matches) == 0 {
		return fmt.Errorf("no process found listening on port %d", port)
	}

	// Use a map to deduplicate PIDs (in case multiple threads show up)
	pids := make(map[string]bool)
	for _, match := range matches {
		pids[match[1]] = true
	}

	// Kill each unique PID found
	for pid := range pids {
		fmt.Printf("-> Found PID %s. Sending SIGTERM...\n", pid)

		killCmd := exec.Command("kill", "-TERM", pid)
		if err := killCmd.Run(); err != nil {
			fmt.Printf("Warning: Failed to kill PID %s: %v\n", pid, err)
		}
	}

	return nil
}

func KillProcByPort(port int) {
	err := killProcByPort(port)
	if err != nil {
		fmt.Printf("❌ Failed to kill process: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("🚀 SIGTERM sent successfully to instance on port 9093.")
}
