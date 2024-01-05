package e2e

import (
	"github.com/stretchr/testify/assert"
	"github.com/vechain/thor/v2/abi"
	"github.com/vechain/thor/v2/thor"
	"math/big"
	"testing"
)

var (
	transferAbi, _ = abi.New([]byte(`[{
			"inputs": [
				{
					"internalType": "address",
					"name": "to",
					"type": "address"
				},
				{
					"internalType": "uint256",
					"name": "value",
					"type": "uint256"
				}
			],
			"name": "transfer",
			"outputs": [
				{
					"internalType": "bool",
					"name": "",
					"type": "bool"
				}
			],
			"stateMutability": "nonpayable",
			"type": "function"
		}
	]`))
)

func encodeFungibleTransfer(t *testing.T, receiver thor.Address, amount *big.Int) ([]byte, error) {
	method, found := transferAbi.MethodByName("transfer")

	if !found {
		t.Errorf("Method not found: %s", "transfer")
	}

	data, err := method.EncodeInput(receiver, amount)
	assert.NoError(t, err, "EncodeInput()")

	return data, nil
}
