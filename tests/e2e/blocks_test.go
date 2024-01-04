package e2e

import (
	"github.com/steinfletcher/apitest"
	"github.com/stretchr/testify/assert"
	"github.com/vechain/thor/v2/api/blocks"
	"github.com/vechain/thor/v2/tests/e2e/endpoints"
	"testing"
)

func TestGetBlock(t *testing.T) {
	block := new(blocks.JSONCollapsedBlock)

	blockNum := int32(1)

	res := apitest.New().
		EnableNetworking().
		Get(endpoints.Node1.GetCompressedBlock(blockNum)).
		Expect(t).
		Status(200).
		End()

	res.JSON(block)

	assert.Equal(t, block.Number, uint32(blockNum), "GetCompressedBlock()")
}
