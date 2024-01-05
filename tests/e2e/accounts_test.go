package e2e

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/steinfletcher/apitest"
	"github.com/stretchr/testify/assert"
	"github.com/vechain/thor/v2/api/accounts"
	"github.com/vechain/thor/v2/genesis"
	"testing"
)

func TestAccountBalance(t *testing.T) {
	acc := new(accounts.Account)

	address := genesis.DevAccounts()[0].Address.String()

	res := apitest.New().
		EnableNetworking().
		Get(node1.getAccount(address)).
		Expect(t).
		Status(200).
		End()

	res.JSON(acc)

	balance, err := acc.Balance.MarshalText()

	assert.NoError(t, err, "MarshalText()")

	balanceHex := hexutil.Encode(balance)

	assert.Equal(t, balanceHex, "0x307831346164663462373332303333346239303030303030")
}
