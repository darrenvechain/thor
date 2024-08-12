// Copyright (c) 2018 The VeChainThor developers

// Distributed under the GNU Lesser General Public License v3.0 software license, see the accompanying
// file LICENSE or <https://www.gnu.org/licenses/lgpl-3.0.html>

package blocks

import (
	"net/http"
	"sort"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/vechain/thor/v2/api/utils"
	"github.com/vechain/thor/v2/bft"
	"github.com/vechain/thor/v2/block"
	"github.com/vechain/thor/v2/chain"
	"github.com/vechain/thor/v2/thor"
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

	txs, err := b.repo.GetBlockTransactions(summary.Header.ID())
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

	stats := CoefStats{
		Ranges: ranges,
		Total:  len(txs),
	}

	coefs := make([]uint8, 0, len(txs))
	coefCount := make(map[uint8]int)
	sum := 0

	for _, t := range txs {
		coef := t.GasPriceCoef()
		coefs = append(coefs, coef)
		coefCount[coef]++
		sum += int(coef)

		for i := range stats.Ranges {
			if coef >= stats.Ranges[i].Min && coef <= stats.Ranges[i].Max {
				stats.Ranges[i].Percent++
				break
			}
		}
	}

	// Calculate percentages
	if stats.Total > 0 {
		for i := range stats.Ranges {
			stats.Ranges[i].Percent = (stats.Ranges[i].Percent * 100) / stats.Total
		}
	}

	stats.GasUsed = summary.Header.GasUsed()
	stats.UnusedGas = summary.Header.GasLimit() - stats.GasUsed

	// Calculate additional metrics
	if len(coefs) > 0 {
		sort.Slice(coefs, func(i, j int) bool { return coefs[i] < coefs[j] })
		stats.Min = coefs[0]
		stats.Max = coefs[len(coefs)-1]
		stats.Median = coefs[len(coefs)/2]
		stats.Average = float64(sum) / float64(len(coefs))

		mode := uint8(0)
		maxCount := 0
		for coef, count := range coefCount {
			if count > maxCount {
				mode = coef
				maxCount = count
			}
		}
		stats.Mode = mode
	}

	return utils.WriteJSON(w, stats)
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
	sub.Path("/{revision}").
		Methods(http.MethodGet).
		Name("blocks_get_block").
		HandlerFunc(utils.WrapHandlerFunc(b.handleGetBlock))
	sub.Path("/{revision}/coefficients").
		Methods(http.MethodGet).
		Name("blocks_get_coefficients").
		HandlerFunc(utils.WrapHandlerFunc(b.handleGetCoefficients))
}
