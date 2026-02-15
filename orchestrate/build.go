package orchestrate

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Build struct {
	rootDir string
}

func NewBuild() (Build, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return Build{}, fmt.Errorf("failed to determine project root: %w", err)
	}
	rootDir := strings.TrimSpace(string(out))
	binaries := []struct {
		Name string
		SHA  string
	}{
		{
			"am_v0_31_0",
			"f33d4897a96da0ecf9c93dcdcd89ae25b42c056f53bff991340ec685c7f0bf0a",
		},
		{
			"am_proto",
			"698094ea606fb992b060026c661b6c7dbdd0f05f20b328f27910b6b778d7ce3e",
		},
	}

	for _, bin := range binaries {
		if _, err := ensure(rootDir, bin.Name, bin.SHA); err != nil {
			return Build{}, err
		}
	}
	return Build{}, err
}

func build(scriptPath string, binaryPath string) error {
	tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("build-%d", time.Now().UnixNano()))
	fmt.Printf("Creating %s \n", tmpDir)
	err := os.Mkdir(tmpDir, 0o700)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	cmd := exec.Command(scriptPath, binaryPath)
	cmd.Dir = tmpDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func ensure(rootDir string, name string, expectedSHA string) (string, error) {
	scriptPath := filepath.Join(rootDir, "scripts", "build", name+".sh")
	binaryPath := filepath.Join(rootDir, "bin", name)

	if _, err := os.Stat(binaryPath); errors.Is(err, os.ErrNotExist) {
		fmt.Printf("Building %s using %s...\n", name, filepath.Base(scriptPath))
		err = build(scriptPath, binaryPath)
		if err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	} else {
		fmt.Printf("Found %v\n", binaryPath)
	}

	fmt.Printf("Computing sha of %v\n", binaryPath)
	sha, err := computeSHA(binaryPath)
	if err != nil {
		return "", err
	}
	if sha != expectedSHA {
		return "", fmt.Errorf("SHA-mismatch: got %v, expected %v", sha, expectedSHA)
	}
	return binaryPath, nil
}

func computeSHA(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}
