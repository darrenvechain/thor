package e2e

import (
	"crypto/ecdsa"
	"errors"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/assert"
	"github.com/vechain/thor/v2/api/transactions"
	"github.com/vechain/thor/v2/genesis"
	"github.com/vechain/thor/v2/tx"
	"math/rand"
	"testing"
	"time"
)

// BuildTransaction builds a transaction from a list of clauses
func buildTransaction(clauses []tx.Clause) (*tx.Transaction, error) {

	bestBlock, err := getExpandedBlock(node1.getCompressedBlock("0"))
	if err != nil {
		return nil, err
	}

	genesisBlock, err := getExpandedBlock(node1.getCompressedBlock("0"))
	if err != nil {
		return nil, err
	}

	builder := new(tx.Builder).
		BlockRef(tx.NewBlockRefFromID(bestBlock.ID)).
		ChainTag(genesisBlock.ID.Bytes()[31]).
		Expiration(uint32(10_000)).
		Gas(uint64(10_000_000)).
		Nonce(rand.Uint64())

	for _, clause := range clauses {
		builder.Clause(&clause)
	}

	return builder.Build(), nil
}

// SignAndEncode signs a transaction and encodes it to hex
func signAndEncode(tx *tx.Transaction, key *ecdsa.PrivateKey) (transactions.RawTx, error) {
	sig, err := crypto.Sign(tx.SigningHash().Bytes(), key)

	if err != nil {
		return transactions.RawTx{}, err
	}

	tx = tx.WithSignature(sig)

	rlpTx, err := rlp.EncodeToBytes(tx)

	if err != nil {
		return transactions.RawTx{}, err
	}

	return transactions.RawTx{Raw: hexutil.Encode(rlpTx)}, nil
}

// SendClauses creates a transaction from a list of clauses, signs it and sends it
func sendClauses(t *testing.T, clauses []tx.Clause, signer genesis.DevAccount) *transactions.SubmitTxResponse {
	unsignedTx, err := buildTransaction(clauses)
	assert.NoError(t, err, "Failed to build the transaction")

	signedTx, err := signAndEncode(unsignedTx, signer.PrivateKey)
	assert.NoError(t, err, "Failed to sign the transaction")

	txResponse, err := postTransaction(node1.postTransaction(), signedTx)
	assert.NoError(t, err, "Failed to send the transaction")

	return txResponse
}

// PollReceipt polls the node for a transaction receipt
func pollReceipt(txID string) (*transactions.Receipt, error) {
	for try := 1; try <= 20; try++ {

		receipt, _ := getTransactionReceipt(node1.getTransactionReceipt(txID))

		if receipt != nil && !receipt.GasPayer.IsZero() {
			return receipt, nil
		}

		// Sleep for 1 second before the next poll
		time.Sleep(1 * time.Second)
	}

	return nil, errors.New("maximum number of tries reached")
}
