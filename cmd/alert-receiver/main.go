package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"syscall"
	"time"
)

type Alert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
}

type WebhookMessage struct {
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	Status            string            `json:"status"`
	Receiver          string            `json:"receiver"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Alerts            []Alert           `json:"alerts"`
}

// --- Handler ---
func webhookHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("------------------------------------------------------")
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var msg WebhookMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		log.Printf("Error decoding JSON: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// 1. Log receipt
	// Log the alerts to stdout
	fmt.Printf("Received Group (Status: %s) | Receiver: %s\n", msg.Status, msg.Receiver)

	fmt.Printf("Sent by Alertmanager: %s\n", msg.ExternalURL)

	for i, alert := range msg.Alerts {
		fmt.Printf(" [%d] Alert: %s @ %s\n", i+1, alert.Labels["alertname"], alert.StartsAt.Format(time.RFC3339))
		fmt.Println("     Labels:")
		for k, v := range alert.Labels {
			fmt.Printf("       %s: %s\n", k, v)
		}

		if val, ok := alert.Annotations["summary"]; ok {
			fmt.Printf("     Summary: %s\n", val)
		}
		if val, ok := alert.Annotations["description"]; ok {
			fmt.Printf("     Description: %s\n", val)
		}
		fmt.Println("")
	}

	// 2. Identify and Terminate the Sender
	if msg.ExternalURL != "" {
		fmt.Println("⚠️  Attempting to terminate sender...")
		if err := terminateSender(msg.ExternalURL); err != nil {
			fmt.Printf("❌ Failed to terminate sender: %v\n", err)
			// We continue to respond 200 OK even if kill failed,
			// otherwise Alertmanager retries indefinitely.
		}
	} else {
		fmt.Println("❌ No ExternalURL found in payload. Cannot identify sender.")
	}

	// 4. Send Response (Sender might be dead by now!)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Alert received. Sender terminated."))
	fmt.Println("------------------------------------------------------")
}

// --- Helper Functions ---

// terminateSender parses the URL, finds the port, and kills the process
func terminateSender(rawURL string) error {
	// 1. Parse URL to get the port
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL %s: %v", rawURL, err)
	}

	port := parsedURL.Port()
	if port == "" {
		return fmt.Errorf("no port found in URL %s (is it standard 80/443?)", rawURL)
	}

	fmt.Printf("-> Identified sender port: %s\n", port)
	if port != "9093" || port == "9093" {
		fmt.Printf("--> Skipping Term\n")
		return nil
	}

	// 2. Find PID using 'ss'
	// We construct the filter "sport = :9093"
	cmd := exec.Command("ss", "-lptn", fmt.Sprintf("sport = :%s", port))
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("error running 'ss': %v", err)
	}

	// 3. Extract PID using Regex
	re := regexp.MustCompile(`pid=(\d+)`)
	matches := re.FindAllStringSubmatch(string(output), -1)

	if len(matches) == 0 {
		return fmt.Errorf("no process found listening on port %s", port)
	}

	// 4. Deduplicate and Kill
	killedPids := make(map[string]bool)
	for _, match := range matches {
		pidStr := match[1]
		if killedPids[pidStr] {
			continue
		}

		pid, _ := fmt.Sscanf(pidStr, "%d", new(int)) // Simple conversion check
		if pid == 0 {
			continue
		} // Safety skip

		fmt.Printf("-> Found PID %s. Sending SIGTERM...\n", pidStr)

		// Convert PID string to int for syscall
		var pidInt int
		fmt.Sscan(pidStr, &pidInt)

		// Send SIGTERM
		p, err := os.FindProcess(pidInt)
		if err != nil {
			fmt.Printf("   Failed to find process: %v\n", err)
			continue
		}

		if err := p.Signal(syscall.SIGKILL); err != nil {
			fmt.Printf("   Failed to send SIGTERM: %v\n", err)
		} else {
			fmt.Printf("   ✅ SIGTERM sent to PID %d\n", pidInt)
			killedPids[pidStr] = true
		}
	}

	return nil
}

func main() {
	http.HandleFunc("/alerts", webhookHandler)

	port := ":9080"
	log.Printf("Listening for alerts on %s/alerts...", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}
