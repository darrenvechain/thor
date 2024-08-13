// Copyright (c) 2018 The VeChainThor developers

// Distributed under the GNU Lesser General Public License v3.0 software license, see the accompanying
// file LICENSE or <https://www.gnu.org/licenses/lgpl-3.0.html>
package node_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/vechain/thor/v2/api/node"
	"github.com/vechain/thor/v2/chain"
	"github.com/vechain/thor/v2/comm"
	"github.com/vechain/thor/v2/genesis"
	"github.com/vechain/thor/v2/muxdb"
	"github.com/vechain/thor/v2/state"
	"github.com/vechain/thor/v2/txpool"
)

var ts *httptest.Server

func TestNode(t *testing.T) {
	info := node.Info{
		Version: "2.0.93",
	}
	initCommServer(t, info)

	peersRes := httpGet(t, ts.URL+"/node/network/peers")
	var peersStats map[string]string
	if err := json.Unmarshal(peersRes, &peersStats); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 0, len(peersStats), "count should be zero")

	infoRes := httpGet(t, ts.URL+"/node/info")
	var infoResMap map[string]interface{}
	if err := json.Unmarshal(infoRes, &infoResMap); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "2.0.93", infoResMap["version"], "version should be 2.0.93")
}

func initCommServer(t *testing.T, info node.Info) {
	db := muxdb.NewMem()
	stater := state.NewStater(db)
	gene := genesis.NewDevnet()

	b, _, _, err := gene.Build(stater)
	if err != nil {
		t.Fatal(err)
	}
	repo, _ := chain.NewRepository(db, b)
	comm := comm.New(repo, txpool.New(repo, stater, txpool.Options{
		Limit:           10000,
		LimitPerAccount: 16,
		MaxLifetime:     10 * time.Minute,
	}))
	router := mux.NewRouter()
	node.New(comm, info).Mount(router, "/node")
	ts = httptest.NewServer(router)
}

func httpGet(t *testing.T, url string) []byte {
	res, err := http.Get(url) // nolint:gosec
	if err != nil {
		t.Fatal(err)
	}
	r, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	return r
}
