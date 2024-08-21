package runtime

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func findAvailablePort(t *testing.T) int {
	// Create a listener on any available port
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	// Extract the port number from the listener's address
	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port
}

func startSolo(t *testing.T, customArgs []string) (string, error) {
	apiAddr := fmt.Sprintf("localhost:%d", findAvailablePort(t))
	dataDir := t.TempDir()
	t.Cleanup(func() {
		_ = os.RemoveAll(dataDir)
	})

	args := os.Args[0:1]
	staticArgs := []string{"solo", "--api-addr", apiAddr, "--persist", "--data-dir", dataDir}
	args = append(args, staticArgs...)
	args = append(args, customArgs...)

	go func() {
		Start(args)
	}()

	ticker := time.NewTicker(100 * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			res, err := http.Get("http://" + apiAddr + "/blocks/0")
			if err == nil && res.StatusCode == http.StatusOK {
				return apiAddr, nil
			}
		case <-time.After(5 * time.Second):
			return "", errors.New("timeout waiting for solo to start")
		}
	}
}

func TestSolo(t *testing.T) {
	url, err := startSolo(t, nil)
	assert.NoError(t, err)

	res, err := http.Get("http://" + url + "/blocks/0")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
}
