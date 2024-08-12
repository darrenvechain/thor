// Copyright (c) 2018 The VeChainThor developers

// Distributed under the GNU Lesser General Public License v3.0 software license, see the accompanying
// file LICENSE or <https://www.gnu.org/licenses/lgpl-3.0.html>

package blocks

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/vechain/thor/v2/api/utils"
	"github.com/vechain/thor/v2/bft"
	"github.com/vechain/thor/v2/block"
	"github.com/vechain/thor/v2/chain"
	"github.com/vechain/thor/v2/thor"
	"github.com/vechain/thor/v2/tx"
)

type Blocks struct {
	repo *chain.Repository
	bft  bft.Finalizer
}

func New(repo *chain.Repository, bft bft.Finalizer) *Blocks {
	return &Blocks{
		repo,
		bft,
	}
}

func (b *Blocks) handleGetBlock(w http.ResponseWriter, req *http.Request) error {
	revision, err := utils.ParseRevision(mux.Vars(req)["revision"], false)
	if err != nil {
		return utils.BadRequest(errors.WithMessage(err, "revision"))
	}
	expanded := req.URL.Query().Get("expanded")
	if expanded != "" && expanded != "false" && expanded != "true" {
		return utils.BadRequest(errors.WithMessage(errors.New("should be boolean"), "expanded"))
	}

	summary, err := utils.GetSummary(revision, b.repo, b.bft)
	if err != nil {
		if b.repo.IsNotFound(err) {
			return utils.WriteJSON(w, nil)
		}
		return err
	}

	isTrunk, err := b.isTrunk(summary.Header.ID(), summary.Header.Number())
	if err != nil {
		return err
	}

	var isFinalized bool
	if isTrunk {
		finalized := b.bft.Finalized()
		if block.Number(finalized) >= summary.Header.Number() {
			isFinalized = true
		}
	}

	jSummary := buildJSONBlockSummary(summary, isTrunk, isFinalized)
	if expanded == "true" {
		txs, err := b.repo.GetBlockTransactions(summary.Header.ID())
		if err != nil {
			return err
		}
		receipts, err := b.repo.GetBlockReceipts(summary.Header.ID())
		if err != nil {
			return err
		}

		return utils.WriteJSON(w, &JSONExpandedBlock{
			jSummary,
			buildJSONEmbeddedTxs(txs, receipts),
		})
	}

	return utils.WriteJSON(w, &JSONCollapsedBlock{
		jSummary,
		summary.Txs,
	})
}

func (b *Blocks) handleGetCoefficients(w http.ResponseWriter, req *http.Request) error {
	revision, err := utils.ParseRevision(mux.Vars(req)["revision"], false)
	if err != nil {
		return utils.BadRequest(errors.WithMessage(err, "revision"))
	}
	summary, err := utils.GetSummary(revision, b.repo, b.bft)
	if err != nil {
		if b.repo.IsNotFound(err) {
			return utils.WriteJSON(w, nil)
		}
		return err
	}

	var txs []*tx.Transaction
	txs, err = b.repo.GetBlockTransactions(summary.Header.ID())
	if err != nil {
		return err
	}

	ranges := []CoefRange{
		{Min: 0, Max: 0},
		{Min: 1, Max: 1<<5 - 1},
		{Min: 1 << 5, Max: 1<<6 - 1},
		{Min: 1 << 6, Max: 1<<7 - 1},
		{Min: 1 << 7, Max: 1<<8 - 1},
	}

	histogram := CoefStats{
		Ranges: ranges,
		Total:  len(txs),
	}

	for _, t := range txs {
		coef := t.GasPriceCoef()
		for i := range histogram.Ranges {
			if coef >= histogram.Ranges[i].Min && coef <= histogram.Ranges[i].Max {
				histogram.Ranges[i].Percent++
				break
			}
		}
	}

	// Calculate percentages
	if histogram.Total > 0 {
		for i := range histogram.Ranges {
			histogram.Ranges[i].Percent = (histogram.Ranges[i].Percent * 100) / histogram.Total
		}
	}

	histogram.GasUsed = summary.Header.GasUsed()
	histogram.UnusedGas = summary.Header.GasLimit() - histogram.GasUsed

	return utils.WriteJSON(w, histogram)
}

func (b *Blocks) isTrunk(blkID thor.Bytes32, blkNum uint32) (bool, error) {
	idByNum, err := b.repo.NewBestChain().GetBlockID(blkNum)
	if err != nil {
		return false, err
	}
	return blkID == idByNum, nil
}

func (b *Blocks) Mount(root *mux.Router, pathPrefix string) {
	sub := root.PathPrefix(pathPrefix).Subrouter()
	sub.Path("/coefficients").
		Methods(http.MethodGet).
		Name("blocks_get_coefficients").
		HandlerFunc(utils.WrapHandlerFunc(b.handleGetCoefficients))
	sub.Path("/{revision}").
		Methods(http.MethodGet).
		Name("blocks_get_block").
		HandlerFunc(utils.WrapHandlerFunc(b.handleGetBlock))
}
