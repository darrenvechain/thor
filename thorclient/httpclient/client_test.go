// Copyright (c) 2024 The VeChainThor developers

// Distributed under the GNU Lesser General Public License v3.0 software license, see the accompanying
// file LICENSE or <https://www.gnu.org/licenses/lgpl-3.0.html>

package httpclient

import (
	"encoding/json"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/vechain/thor/v2/api"
	"github.com/vechain/thor/v2/api/accounts"
	"github.com/vechain/thor/v2/api/blocks"
	"github.com/vechain/thor/v2/api/events"
	"github.com/vechain/thor/v2/api/node"
	"github.com/vechain/thor/v2/api/transactions"
	"github.com/vechain/thor/v2/api/transfers"
	"github.com/vechain/thor/v2/block"
	"github.com/vechain/thor/v2/chain"
	"github.com/vechain/thor/v2/cmd/thor/solo"
	"github.com/vechain/thor/v2/co"
	"github.com/vechain/thor/v2/genesis"
	"github.com/vechain/thor/v2/logdb"
	"github.com/vechain/thor/v2/muxdb"
	"github.com/vechain/thor/v2/packer"
	"github.com/vechain/thor/v2/state"
	"github.com/vechain/thor/v2/thor"
	"github.com/vechain/thor/v2/thorclient/common"
	"github.com/vechain/thor/v2/tx"
	"github.com/vechain/thor/v2/txpool"
)

var (
	genesisBlock *block.Block
	apiURL       = "http://localhost:0"
	transaction  *tx.Transaction
	mempoolTx    *tx.Transaction
)

func initTransaction(repo *chain.Repository, stater *state.Stater, b *block.Block) (*tx.Transaction, error) {
	addr := thor.BytesToAddress([]byte("to"))
	cla := tx.NewClause(&addr).WithValue(big.NewInt(10000))
	transaction = new(tx.Builder).
		ChainTag(repo.ChainTag()).
		GasPriceCoef(1).
		Expiration(10).
		Gas(21000).
		Nonce(1).
		Clause(cla).
		BlockRef(tx.NewBlockRef(0)).
		Build()

	mempoolTx = new(tx.Builder).
		ChainTag(repo.ChainTag()).
		Expiration(10).
		Gas(21000).
		Nonce(1).
		Build()

	sig, err := crypto.Sign(transaction.SigningHash().Bytes(), genesis.DevAccounts()[0].PrivateKey)
	if err != nil {
		return nil, err
	}

	sig2, err := crypto.Sign(mempoolTx.SigningHash().Bytes(), genesis.DevAccounts()[0].PrivateKey)
	if err != nil {
		return nil, err
	}

	transaction = transaction.WithSignature(sig)
	mempoolTx = mempoolTx.WithSignature(sig2)

	packer := packer.New(repo, stater, genesis.DevAccounts()[0].Address, &genesis.DevAccounts()[0].Address, thor.NoFork)
	sum, _ := repo.GetBlockSummary(b.Header().ID())
	flow, err := packer.Schedule(sum, uint64(time.Now().Unix()))
	if err != nil {
		return nil, err
	}
	err = flow.Adopt(transaction)
	if err != nil {
		return nil, err
	}
	b, stage, receipts, err := flow.Pack(genesis.DevAccounts()[0].PrivateKey, 0, false)
	if err != nil {
		return nil, err
	}
	if _, err := stage.Commit(); err != nil {
		return nil, err
	}
	if err := repo.AddBlock(b, receipts, 0); err != nil {
		return nil, err
	}
	if err := repo.SetBestBlockID(b.Header().ID()); err != nil {
		return nil, err
	}

	return mempoolTx, nil
}

func TestMain(m *testing.M) {
	db := muxdb.NewMem()
	stater := state.NewStater(db)
	gene := genesis.NewDevnet()
	logDB, err := logdb.NewMem()
	if err != nil {
		panic(err)
	}
	genesisBlock, _, _, err = gene.Build(stater)
	if err != nil {
		panic(err)
	}
	repo, _ := chain.NewRepository(db, genesisBlock)

	mempoolTx, err := initTransaction(repo, stater, genesisBlock)
	if err != nil {
		panic(err)
	}

	mempool := txpool.New(
		repo,
		stater,
		txpool.Options{Limit: 10000, LimitPerAccount: 16, MaxLifetime: 10 * time.Minute},
	)
	if err := mempool.Add(mempoolTx); err != nil {
		panic(err)
	}
	bft := solo.NewBFTEngine(repo)
	handler, closer := api.New(
		repo,
		stater,
		mempool,
		logDB,
		bft,
		&solo.Communicator{},
		thor.NoFork,
		"*",
		5,
		10_000_000,
		false,
		false,
		true,
		false,
		false,
		1000)
	defer closer()

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}
	apiURL = "http://" + listener.Addr().String()
	defer listener.Close()
	srv := &http.Server{Handler: handler, ReadHeaderTimeout: time.Second, ReadTimeout: 5 * time.Second}
	defer srv.Close()
	var goes co.Goes
	goes.Go(func() {
		srv.Serve(listener)
	})

	m.Run()
}

func TestClient_GetTransactionReceipt(t *testing.T) {
	client := New(apiURL)
	txID := transaction.ID()
	receipt, err := client.GetTransactionReceipt(&txID)

	assert.NoError(t, err)
	assert.Equal(t, transaction.ID(), receipt.Meta.TxID)
}

func TestClient_InspectClauses(t *testing.T) {
	calldata := &accounts.BatchCallData{}
	expectedResults := []*accounts.CallResult{{
		Data:      "data",
		Events:    []*transactions.Event{},
		Transfers: []*transactions.Transfer{},
		GasUsed:   1000,
		Reverted:  false,
		VMError:   "no error"}}

	client := New(apiURL)
	results, err := client.InspectClauses(calldata)

	assert.NoError(t, err)
	assert.Equal(t, expectedResults, results)
}

func TestClient_SendTransaction(t *testing.T) {
	rawTx := &transactions.RawTx{}
	expectedResult := &common.TxSendResult{ID: &thor.Bytes32{0x01}}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/transactions", r.URL.Path)

		txIDBytes, _ := json.Marshal(expectedResult)
		w.Write(txIDBytes)
	}))
	defer ts.Close()

	client := New(ts.URL)
	result, err := client.SendTransaction(rawTx)

	assert.NoError(t, err)
	assert.Equal(t, expectedResult, result)
}

func TestClient_FilterTransfers(t *testing.T) {
	req := &events.EventFilter{}
	expectedTransfers := []*transfers.FilteredTransfer{{
		Sender:    thor.Address{0x01},
		Recipient: thor.Address{0x02},
		Amount:    &math.HexOrDecimal256{},
		Meta:      transfers.LogMeta{},
	}}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/logs/transfer", r.URL.Path)

		filteredTransfersBytes, _ := json.Marshal(expectedTransfers)
		w.Write(filteredTransfersBytes)
	}))
	defer ts.Close()

	client := New(ts.URL)
	transfers, err := client.FilterTransfers(req)

	assert.NoError(t, err)
	assert.Equal(t, expectedTransfers, transfers)
}

func TestClient_FilterEvents(t *testing.T) {
	req := &events.EventFilter{}
	expectedEvents := []events.FilteredEvent{{
		Address: thor.Address{0x01},
		Topics:  []*thor.Bytes32{{0x01}},
		Data:    "data",
		Meta:    events.LogMeta{},
	}}
	expectedPath := "/logs/event"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, expectedPath, r.URL.Path)

		filteredEventsBytes, _ := json.Marshal(expectedEvents)
		w.Write(filteredEventsBytes)
	}))
	defer ts.Close()

	client := New(ts.URL)
	events, err := client.FilterEvents(req)

	assert.NoError(t, err)
	assert.Equal(t, expectedEvents, events)
}

func TestClient_GetAccount(t *testing.T) {
	addr := thor.Address{0x01}
	expectedAccount := &accounts.Account{
		Balance: math.HexOrDecimal256{},
		Energy:  math.HexOrDecimal256{},
		HasCode: false,
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/accounts/"+addr.String(), r.URL.Path)

		accountBytes, _ := json.Marshal(expectedAccount)
		w.Write(accountBytes)
	}))
	defer ts.Close()

	client := New(ts.URL)
	account, err := client.GetAccount(&addr, "")

	assert.NoError(t, err)
	assert.Equal(t, expectedAccount, account)
}

func TestClient_GetAccountCode(t *testing.T) {
	addr := thor.Address{0x01}
	// expected is a map with "code" as the only key
	expectedByteCode := []byte{0x01}
	expected := map[string]string{"code": ethcommon.Bytes2Hex(expectedByteCode)}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/accounts/"+addr.String()+"/code", r.URL.Path)

		accountBytes, _ := json.Marshal(expected)
		w.Write(accountBytes)
	}))
	defer ts.Close()

	client := New(ts.URL)
	byteCode, err := client.GetAccountCode(&addr, "")

	assert.NoError(t, err)
	assert.Equal(t, expectedByteCode, byteCode)
}

func TestClient_GetStorage(t *testing.T) {
	addr := thor.Address{0x01}
	key := thor.Bytes32{0x01}
	expectedData := []byte{0x01}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/accounts/"+addr.String()+"/key/"+key.String(), r.URL.Path)

		w.Write(expectedData)
	}))
	defer ts.Close()

	client := New(ts.URL)
	data, err := client.GetStorage(&addr, &key)

	assert.NoError(t, err)
	assert.Equal(t, expectedData, data)
}

func TestClient_GetExpandedBlock(t *testing.T) {
	blockID := "123"
	expectedBlock := &blocks.JSONExpandedBlock{}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/blocks/"+blockID+"?expanded=true", r.URL.Path+"?"+r.URL.RawQuery)

		blockBytes, _ := json.Marshal(expectedBlock)
		w.Write(blockBytes)
	}))
	defer ts.Close()

	client := New(ts.URL)
	block, err := client.GetBlockExpanded(blockID)

	assert.NoError(t, err)
	assert.Equal(t, expectedBlock, block)
}

func TestClient_GetBlock(t *testing.T) {
	blockID := "123"
	expectedBlock := &blocks.JSONBlockSummary{
		Number:      123456,
		ID:          thor.Bytes32{0x01},
		GasLimit:    1000,
		Beneficiary: thor.Address{0x01},
		GasUsed:     100,
		TxsRoot:     thor.Bytes32{0x03},
		TxsFeatures: 1,
		IsFinalized: false,
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/blocks/"+blockID, r.URL.Path)

		blockBytes, _ := json.Marshal(expectedBlock)
		w.Write(blockBytes)
	}))
	defer ts.Close()

	client := New(ts.URL)
	block, err := client.GetBlock(blockID)

	assert.NoError(t, err)
	assert.Equal(t, expectedBlock, block)
}

func TestClient_GetTransaction(t *testing.T) {
	txID := thor.Bytes32{0x01}
	expectedTx := &transactions.Transaction{ID: txID}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/transactions/"+txID.String(), r.URL.Path)

		txBytes, _ := json.Marshal(expectedTx)
		w.Write(txBytes)
	}))
	defer ts.Close()

	client := New(ts.URL)
	tx, err := client.GetTransaction(&txID, false, false)

	assert.NoError(t, err)
	assert.Equal(t, expectedTx, tx)
}

func TestClient_RawHTTPPost(t *testing.T) {
	url := "/test"
	calldata := map[string]interface{}{}
	expectedResponse := []byte{0x01}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, url, r.URL.Path)

		w.Write(expectedResponse)
	}))
	defer ts.Close()

	client := New(ts.URL)
	response, statusCode, err := client.RawHTTPPost(url, calldata)

	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, response)
	assert.Equal(t, http.StatusOK, statusCode)
}

func TestClient_RawHTTPGet(t *testing.T) {
	url := "/test"
	expectedResponse := []byte{0x01}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, url, r.URL.Path)

		w.Write(expectedResponse)
	}))
	defer ts.Close()

	client := New(ts.URL)
	response, statusCode, err := client.RawHTTPGet(url)

	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, response)
	assert.Equal(t, http.StatusOK, statusCode)
}

func TestClient_GetPeers(t *testing.T) {
	expectedPeers := []*node.PeerStats{{
		Name:        "nodeA",
		BestBlockID: thor.Bytes32{0x01},
		TotalScore:  1000,
		PeerID:      "peerId",
		NetAddr:     "netAddr",
		Inbound:     false,
		Duration:    1000,
	}}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/node/network/peers", r.URL.Path)

		peersBytes, _ := json.Marshal(expectedPeers)
		w.Write(peersBytes)
	}))
	defer ts.Close()

	client := New(ts.URL)
	peers, err := client.GetPeers()

	assert.NoError(t, err)
	assert.Equal(t, expectedPeers, peers)
}

func TestClient_Errors(t *testing.T) {
	txID := thor.Bytes32{0x01}
	blockID := "123"
	addr := thor.Address{0x01}

	for _, tc := range []struct {
		name     string
		path     string
		function interface{}
	}{
		{
			name:     "GetTransactionReceipt",
			path:     "/transactions/" + txID.String() + "/receipt",
			function: func(client *Client) (*transactions.Receipt, error) { return client.GetTransactionReceipt(&txID) },
		},
		{
			name: "InspectClauses",
			path: "/accounts/*",
			function: func(client *Client) ([]*accounts.CallResult, error) {
				return client.InspectClauses(&accounts.BatchCallData{})
			},
		},
		{
			name: "SendTransaction",
			path: "/transactions",
			function: func(client *Client) (*common.TxSendResult, error) {
				return client.SendTransaction(&transactions.RawTx{})
			},
		},
		{
			name: "FilterTransfers",
			path: "/logs/transfer",
			function: func(client *Client) ([]*transfers.FilteredTransfer, error) {
				return client.FilterTransfers(&events.EventFilter{})
			},
		},
		{
			name: "FilterEvents",
			path: "/logs/event",
			function: func(client *Client) ([]events.FilteredEvent, error) {
				return client.FilterEvents(&events.EventFilter{})
			},
		},
		{
			name:     "GetAccount",
			path:     "/accounts/" + addr.String(),
			function: func(client *Client) (*accounts.Account, error) { return client.GetAccount(&addr, "") },
		},
		{
			name:     "GetContractByteCode",
			path:     "/accounts/" + addr.String() + "/code",
			function: func(client *Client) ([]byte, error) { return client.GetAccountCode(&addr, "") },
		},
		{
			name:     "GetStorage",
			path:     "/accounts/" + addr.String() + "/key/" + thor.Bytes32{}.String(),
			function: func(client *Client) ([]byte, error) { return client.GetStorage(&addr, &thor.Bytes32{}) },
		},
		{
			name:     "GetBlockExpanded",
			path:     "/blocks/" + blockID + "?expanded=true",
			function: func(client *Client) (*blocks.JSONExpandedBlock, error) { return client.GetBlockExpanded(blockID) },
		},
		{
			name:     "GetBlock",
			path:     "/blocks/" + blockID,
			function: func(client *Client) (*blocks.JSONBlockSummary, error) { return client.GetBlock(blockID) },
		},
		{
			name: "GetTransaction",
			path: "/transactions/" + txID.String(),
			function: func(client *Client) (*transactions.Transaction, error) {
				return client.GetTransaction(&txID, false, false)
			},
		},
		{
			name:     "GetPeers",
			path:     "/node/network/peers",
			function: func(client *Client) ([]*node.PeerStats, error) { return client.GetPeers() },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, tc.path, r.URL.Path)

				w.WriteHeader(http.StatusInternalServerError)
			}))
			defer ts.Close()

			client := New(ts.URL)

			fn := reflect.ValueOf(tc.function)
			result := fn.Call([]reflect.Value{reflect.ValueOf(client)})

			if result[len(result)-1].IsNil() {
				t.Errorf("expected error for %s, but got nil", tc.name)
				return
			}

			err := result[len(result)-1].Interface().(error)
			assert.Error(t, err)
		})
	}
}
