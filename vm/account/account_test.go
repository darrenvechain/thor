package account

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vechain/thor/acc"
	"github.com/vechain/thor/cry"
)

func TestAccount_deepCopy(t *testing.T) {
	assert := assert.New(t)

	account1 := newAccount(acc.Address{1}, new(big.Int))
	account1.DirtyStorage[cry.Hash{1}] = cry.Hash{200}
	account1.cachedStorage[cry.Hash{1}] = cry.Hash{200}

	account2 := account1.deepCopy()
	assertAccount(assert, account1, account2)
}

func assertAccount(assert *assert.Assertions, account1 *Account, account2 *Account) {
	assert.Equal(account1, account2, "未改值前应该相等.")

	account1.Balance.SetInt64(100)
	assert.NotEqual(account1.Balance, account2.Balance, "修改了 Balance, 应该不相等.")

	account1.Address = acc.Address{2}
	assert.NotEqual(account1.Address, account2.Address, "修改了 address, 应该不相等.")

	account1.DirtyStorage[cry.Hash{1}] = cry.Hash{100}
	assert.NotEqual(account1.DirtyStorage, account2.DirtyStorage, "修改了 Storage, 应该不相等.")

	account1.cachedStorage[cry.Hash{1}] = cry.Hash{100}
	assert.NotEqual(account1.cachedStorage, account2.cachedStorage, "修改了 cachedStorage, 应该不相等.")
}