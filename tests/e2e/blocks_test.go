package e2e

import (
	"fmt"
	"github.com/steinfletcher/apitest"
	"github.com/stretchr/testify/assert"
	"github.com/vechain/thor/v2/api/blocks"
	"testing"
)

func TestGetBlock(t *testing.T) {
	block := new(blocks.JSONCollapsedBlock)

	blockNum := uint32(1)
	blockRevision := fmt.Sprintf("%d", blockNum)

	res := apitest.New().
		EnableNetworking().
		Get(node1.getCompressedBlock(blockRevision)).
		Expect(t).
		Status(200).
		End()

	res.JSON(block)

	assert.Equal(t, blockNum, block.Number, "GetCompressedBlock()")
}
