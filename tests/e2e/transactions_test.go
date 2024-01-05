package e2e

import (
	"github.com/stretchr/testify/assert"
	"github.com/vechain/thor/v2/genesis"
	"github.com/vechain/thor/v2/thor"
	"github.com/vechain/thor/v2/tx"
	"math/big"
	"testing"
)

func TestSendVetTransaction(t *testing.T) {
	originAccount := genesis.DevAccounts()[1]
	toAccount := genesis.DevAccounts()[2]

	clause := tx.NewClause(&toAccount.Address).WithValue(big.NewInt(1e18))
	res := sendClauses(t, []tx.Clause{*clause}, originAccount)
	receipt, err := pollReceipt(res.ID)

	assert.NoError(t, err, "Failed to poll receipt")
	assert.Equal(t, false, receipt.Reverted, "Transaction reverted")
}

func TestSendVthoTransaction(t *testing.T) {
	originAccount := genesis.DevAccounts()[1]
	toAccount := genesis.DevAccounts()[2]

	vtho, _ := thor.ParseAddress("0x0000000000000000000000000000456E65726779")
	data, err := encodeFungibleTransfer(t, toAccount.Address, big.NewInt(1e18))
	clause := tx.NewClause(&vtho).WithData(data)
	res := sendClauses(t, []tx.Clause{*clause}, originAccount)
	receipt, err := pollReceipt(res.ID)

	assert.NoError(t, err, "Failed to poll receipt")
	assert.Equal(t, false, receipt.Reverted, "Transaction reverted")
}
