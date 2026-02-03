package orchestrate

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// ANSI Colors for terminal output
var colors = []string{
	"\033[36m", // Cyan
	"\033[32m", // Green
	"\033[33m", // Yellow
	"\033[35m", // Magenta
	"\033[31m", // Red
	"\033[34m", // Blue
}

const colorReset = "\033[0m"

type Instance struct {
	Name        string
	WebPort     int
	ClusterPort int
}

func StartReceiver() (*exec.Cmd, error) {
	cwd, _ := os.Getwd()
	binaryPath := filepath.Join(cwd, "bin", "alert-receiver")

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("receiver binary not found at %s", binaryPath)
	}

	cmd := exec.Command(binaryPath)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	prefix := fmt.Sprintf("%s[Receiver]%s ", colors[4], colorReset)

	go streamLog(prefix, stdout)
	go streamLog(prefix, stderr)

	fmt.Printf("Starting Receiver on port %d...\n", 9080)

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

func StartLocalCluster(setupName string, count int) ([]*exec.Cmd, error) {
	cwd, _ := os.Getwd()
	binaryPath := filepath.Join(cwd, "bin", "alertmanager")
	configPath := filepath.Join(cwd, "setups", setupName, "alertmanager.yml")

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("binary not found at %s", binaryPath)
	}

	instances := []Instance{
		{Name: "01-zebra", WebPort: 9093, ClusterPort: 9094},
		{Name: "02-lion", WebPort: 9095, ClusterPort: 9096},
		{Name: "03-tiger", WebPort: 9097, ClusterPort: 9098},
	}

	var runningCmds []*exec.Cmd
	bootstrapPeer := fmt.Sprintf("127.0.0.1:%d", instances[0].ClusterPort)

	for i := range count {
		inst := instances[i]
		tempStorage, err := os.MkdirTemp("", fmt.Sprintf("am-storage-%s-", inst.Name))
		if err != nil {
			return nil, err
		}

		args := []string{
			fmt.Sprintf("--config.file=%s", configPath),
			fmt.Sprintf("--storage.path=%s", tempStorage),
			fmt.Sprintf("--web.listen-address=127.0.0.1:%d", inst.WebPort),
			fmt.Sprintf("--cluster.listen-address=127.0.0.1:%d", inst.ClusterPort),
			fmt.Sprintf("--cluster.peer-name=%s", inst.Name),
			"--cluster.gossip-interval=200ms",
			"--cluster.pushpull-interval=1m",
			"--log.level=info",
		}

		if i > 0 {
			args = append(args, fmt.Sprintf("--cluster.peer=%s", bootstrapPeer))
		}

		cmd := exec.Command(binaryPath, args...)

		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()

		myColor := colors[i%len(colors)]
		prefix := fmt.Sprintf("%s[%s]%s ", myColor, inst.Name, colorReset)

		go streamLog(prefix, stdout)
		go streamLog(prefix, stderr)

		fmt.Printf("Starting %s on port %d...\n", inst.Name, inst.WebPort)

		if err := cmd.Start(); err != nil {
			return nil, err
		}
		runningCmds = append(runningCmds, cmd)
	}

	return runningCmds, nil
}

func streamLog(prefix string, rc io.ReadCloser) {
	scanner := bufio.NewScanner(rc)
	for scanner.Scan() {
		fmt.Println(prefix + scanner.Text())
	}
}

type Alert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    time.Time         `json:"startsAt"`
	EndsAt      time.Time         `json:"endsAt"`
}

func SendAlert(payload []Alert, port int) {
	url := fmt.Sprintf("http://localhost:%d/api/v2/alerts", port)
	prefix := fmt.Sprintf("%s[Orchestrator]%s ", colors[5], colorReset)

	jsonBytes, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		fmt.Printf("%s !!! Failed to send alert: %v\n", prefix, err)
		return
	}
	resp.Body.Close()

	fmt.Printf("%s Alerts sent to :%d \n", prefix, port)
}
