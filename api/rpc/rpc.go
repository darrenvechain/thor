package rpc

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/vechain/thor/v2/api/utils"
	"github.com/vechain/thor/v2/chain"
	"github.com/vechain/thor/v2/txpool"
	"net/http"
)

type RPC struct {
	repo *chain.Repository
	pool *txpool.TxPool
}

func New(repo *chain.Repository, pool *txpool.TxPool) *RPC {
	return &RPC{
		repo,
		pool,
	}
}

func (r *RPC) handleEthSendRawTransaction(w http.ResponseWriter, req *http.Request, request rpcRequest) error {
	data, ok := request.Params[0].(string)
	if !ok {
		return utils.BadRequest(errors.New("params[0] is not a string"))
	}

	rawTx, err := hexutil.Decode(data)
	if err != nil {
		return utils.BadRequest(errors.New("invalid raw transaction"))
	}

	var tx types.Transaction
	err = tx.UnmarshalJSON(rawTx)
	if err != nil {
		return utils.BadRequest(errors.WithMessage(err, "failed to unmarshal transaction"))
	}

	tx.Hash()

	return nil
}

func (r *RPC) handleRpcRequest(w http.ResponseWriter, req *http.Request) error {
	rpcRequest := &rpcRequest{}

	if err := utils.ParseJSON(req.Body, rpcRequest); err != nil {
		return utils.BadRequest(errors.WithMessage(err, "params"))
	}

	switch rpcRequest.Method {
	case "eth_sendRawTransaction":
		return r.handleEthSendRawTransaction(w, req, *rpcRequest)
	}

	return utils.BadRequest(errors.Errorf("method %s not found", rpcRequest.Method))
}

func (r *RPC) Mount(root *mux.Router, pathPrefix string) {
	sub := root.PathPrefix(pathPrefix).Subrouter()

	sub.Path("").Methods("POST").HandlerFunc(utils.WrapHandlerFunc(r.handleRpcRequest))
}
