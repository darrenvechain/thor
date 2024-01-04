package e2e

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"testing"
)

// TestMain function is the entry point for the test suite.
// It starts the docker-compose network and stops it after the tests are done.
func TestMain(m *testing.M) {
	// Setup
	err := startCompose()
	if err != nil {
		fmt.Println("failed to start docker compose", err)
		os.Exit(1)
	} else {
		fmt.Println("docker-compose started")
	}

	// Run tests
	exitCode := m.Run()

	// Teardown
	err = stopCompose()

	if err != nil {
		fmt.Println("failed to stop docker compose", err)
		os.Exit(1)
	} else {
		fmt.Println("docker-compose stopped")
	}

	os.Exit(exitCode)
}

func startCompose() error {
	cmd := exec.Command("docker", "compose", "-f", "network/docker-compose.yaml", "up", "-d", "--wait", "--build")

	return executeCommand(cmd)
}

func stopCompose() error {
	cmd := exec.Command("docker", "compose", "-f", "network/docker-compose.yaml", "down", "--remove-orphans", "--rmi", "local")

	return executeCommand(cmd)
}

func executeCommand(cmd *exec.Cmd) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Error creating StdoutPipe: %s", err)
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("Error creating StderrPipe: %s", err)
		return err
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Error starting command: %s", err)
		return err
	}

	go printOutput(stdout)
	go printOutput(stderr)

	if err := cmd.Wait(); err != nil {
		log.Printf("Error waiting for command: %s", err)
		return err
	}

	return nil
}

func printOutput(pipe io.Reader) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
}
