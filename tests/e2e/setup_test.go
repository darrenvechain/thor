package e2e

import (
	"context"
	"fmt"
	"github.com/testcontainers/testcontainers-go/modules/compose"
	"os"
	"testing"
)

// TestMain function is the entry point for the test suite.
// It starts the docker-compose network and stops it after the tests are done.
func TestMain(m *testing.M) {
	// Setup
	err := startCompose()
	if err != nil {
		fmt.Println("failed to start docker-compose", err)
		os.Exit(1)
	} else {
		fmt.Println("docker-compose started")
	}

	// Run tests
	exitCode := m.Run()

	// Teardown
	err = stopCompose()

	if err != nil {
		fmt.Println("failed to stop docker-compose", err)
		os.Exit(1)
	} else {
		fmt.Println("docker-compose stopped")
	}

	os.Exit(exitCode)
}

func startCompose() error {
	dc, err := getComposeStack()

	if err != nil {
		return err
	}

	ctx, _ := context.WithCancel(context.Background())
	err = dc.Up(ctx, compose.Wait(true))

	if err != nil {
		return err
	}

	return nil
}

func stopCompose() error {
	dc, err := getComposeStack()

	if err != nil {
		return err
	}

	ctx, _ := context.WithCancel(context.Background())
	err = dc.Down(ctx, compose.RemoveOrphans(true), compose.RemoveImagesLocal)

	if err != nil {
		return err
	}

	return nil
}

func getComposeStack() (compose.ComposeStack, error) {
	dc, err := compose.NewDockerCompose("network/docker-compose.yaml")

	if err != nil {
		return nil, err
	}

	return dc, nil
}
