// Copyright (c) 2024 The VeChainThor developers

// Distributed under the GNU Lesser General Public License v3.0 software license, see the accompanying
// file LICENSE or <https://www.gnu.org/licenses/lgpl-3.0.html>

package thorclient

import (
	"encoding/json"
	"fmt"

	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vechain/thor/v2/api/accounts"
	"github.com/vechain/thor/v2/api/blocks"
	"github.com/vechain/thor/v2/api/transactions"
	"github.com/vechain/thor/v2/cmd/thor/runtime"
	"github.com/vechain/thor/v2/thor"
	"github.com/vechain/thor/v2/tx"
)

var thorURL string

func newApiAddr() string {
	// Create a listener on any available port
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(fmt.Sprintf("failed to create listener: %v", err))
	}
	defer listener.Close()

	// Extract the port number from the listener's address
	addr := listener.Addr().(*net.TCPAddr)
	return addr.String()
}

func waitForServer(apiAddr string) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	timeout := time.NewTimer(5 * time.Second)

	for {
		select {
		case <-ticker.C:
			// Wait for the server to start
			res, err := http.Get(apiAddr + "/blocks/0")
			if err != nil || res.StatusCode != http.StatusOK {
				continue
			}

			body, err := io.ReadAll(res.Body)
			if err != nil {
				continue
			}

			var block blocks.JSONCollapsedBlock
			if err := json.Unmarshal(body, &block); err == nil {
				return nil
			}
		case <-timeout.C:
			return nil
		}
	}
}

func TestMain(m *testing.M) {
	addr := newApiAddr()
	thorURL = "http://" + addr

	go func() {
		args := []string{
			os.Args[0],
			"solo",
			"--api-addr", addr,
		}
		runtime.Start(args)
	}()

	if err := waitForServer(thorURL); err != nil {
		fmt.Println("failed to start server")
		os.Exit(1)
	}

	m.Run()
}

func TestClient_RealServer(t *testing.T) {
	client := New(thorURL)
	expanded, err := client.ExpandedBlock("0")
	assert.NoError(t, err)
	assert.NotNil(t, expanded)
}

func TestWs_Error(t *testing.T) {
	client := New("http://test.com")

	for _, tc := range []struct {
		name     string
		function interface{}
	}{
		{
			name:     "SubscribeBlocks",
			function: client.SubscribeBlocks,
		},
		{
			name:     "SubscribeEvents",
			function: client.SubscribeEvents,
		},
		{
			name:     "SubscribeTransfers",
			function: client.SubscribeTransfers,
		},
		{
			name:     "SubscribeTxPool",
			function: client.SubscribeTxPool,
		},
		{
			name:     "SubscribeBeats2",
			function: client.SubscribeBeats2,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := reflect.ValueOf(tc.function)
			result := fn.Call([]reflect.Value{})

			if result[1].IsNil() {
				t.Errorf("expected error for %s, but got nil", tc.name)
				return
			}

			err := result[1].Interface().(error)
			assert.Error(t, err)
		})
	}
}

func TestConvertToBatchCallData(t *testing.T) {
	// Test case 1: Empty transaction
	tx1 := &tx.Transaction{}
	addr1 := &thor.Address{}
	expected1 := &accounts.BatchCallData{
		Clauses:    make(accounts.Clauses, 0),
		Gas:        0,
		ProvedWork: nil,
		Caller:     addr1,
		GasPayer:   nil,
		Expiration: 0,
		BlockRef:   "0x0000000000000000",
	}
	assert.Equal(t, expected1, convertToBatchCallData(tx1, addr1))
}

func TestRevision(t *testing.T) {
	addr := thor.BytesToAddress([]byte("account1"))
	revision := "revision1"

	for _, tc := range []struct {
		name             string
		function         interface{}
		expectedPath     string
		expectedRevision string
	}{
		{
			name:             "Account",
			function:         func(client *Client) { client.Account(&addr) },
			expectedPath:     "/accounts/" + addr.String(),
			expectedRevision: "",
		},
		{
			name:             "GetAccounForRevision",
			function:         func(client *Client) { client.Account(&addr, Revision(revision)) },
			expectedPath:     "/accounts/" + addr.String(),
			expectedRevision: "",
		},
		{
			name:             "GetAccountCode",
			function:         func(client *Client) { client.AccountCode(&addr) },
			expectedPath:     "/accounts/" + addr.String() + "/code",
			expectedRevision: "",
		},
		{
			name:             "GetAccountCodeForRevision",
			function:         func(client *Client) { client.AccountCode(&addr, Revision(revision)) },
			expectedPath:     "/accounts/" + addr.String() + "/code",
			expectedRevision: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tc.expectedPath, r.URL.Path)
				if tc.expectedRevision != "" {
					assert.Equal(t, "revision", r.URL.Query().Get("revision"))
				}

				w.Write([]byte{})
			}))
			defer ts.Close()

			client := New(ts.URL)

			fn := reflect.ValueOf(tc.function)
			fn.Call([]reflect.Value{reflect.ValueOf(client)})
		})
	}
}

func TestGetTransaction(t *testing.T) {
	expectedTx := &transactions.Transaction{
		ID: thor.BytesToBytes32([]byte("txid1")),
	}

	for _, tc := range []struct {
		name      string
		function  interface{}
		isPending bool
	}{
		{
			name:      "Transaction",
			function:  func(client *Client) { client.Transaction(&expectedTx.ID) },
			isPending: false,
		},
		{
			name:      "GetTransactionPending",
			function:  func(client *Client) { client.Transaction(&expectedTx.ID, Revision("best"), Pending()) },
			isPending: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/transactions/"+expectedTx.ID.String(), r.URL.Path)
				if tc.isPending {
					assert.Equal(t, "true", r.URL.Query().Get("pending"))
				}

				w.Write(expectedTx.ID[:])
			}))
			defer ts.Close()

			client := New(ts.URL)
			fn := reflect.ValueOf(tc.function)
			fn.Call([]reflect.Value{reflect.ValueOf(client)})
		})
	}
}
