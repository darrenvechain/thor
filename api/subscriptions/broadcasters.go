package subscriptions

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/vechain/thor/v2/chain"
	"github.com/vechain/thor/v2/thor"
	"github.com/vechain/thor/v2/thor/bloom"
)

func createBlockMessage(_ *chain.Repository, block *chain.ExtendedBlock) (interface{}, error) {
	return convertBlock(block)
}

func createBeat2Message(repo *chain.Repository, block *chain.ExtendedBlock) (interface{}, error) {
	bloomGenerator := &bloom.Generator{}

	bloomAdd := func(key []byte) {
		key = bytes.TrimLeft(key, "\x00")
		// exclude non-address key
		if len(key) <= thor.AddressLength {
			bloomGenerator.Add(key)
		}
	}

	header := block.Header()
	receipts, err := repo.GetBlockReceipts(header.ID())
	if err != nil {
		return nil, err
	}
	txs := block.Transactions()
	for i, receipt := range receipts {
		bloomAdd(receipt.GasPayer.Bytes())
		for _, output := range receipt.Outputs {
			for _, event := range output.Events {
				bloomAdd(event.Address.Bytes())
				for _, topic := range event.Topics {
					bloomAdd(topic.Bytes())
				}
			}
			for _, transfer := range output.Transfers {
				bloomAdd(transfer.Sender.Bytes())
				bloomAdd(transfer.Recipient.Bytes())
			}
		}
		origin, _ := txs[i].Origin()
		bloomAdd(origin.Bytes())
	}
	signer, _ := header.Signer()
	bloomAdd(signer.Bytes())
	bloomAdd(header.Beneficiary().Bytes())

	const bitsPerKey = 20
	filter := bloomGenerator.Generate(bitsPerKey, bloom.K(bitsPerKey))

	return &Beat2Message{
		Number:      header.Number(),
		ID:          header.ID(),
		ParentID:    header.ParentID(),
		Timestamp:   header.Timestamp(),
		TxsFeatures: uint32(header.TxsFeatures()),
		GasLimit:    header.GasLimit(),
		Bloom:       hexutil.Encode(filter.Bits),
		K:           filter.K,
		Obsolete:    block.Obsolete,
	}, nil
}

func createBeatMessage(repo *chain.Repository, block *chain.ExtendedBlock) (interface{}, error) {
	header := block.Header()
	receipts, err := repo.GetBlockReceipts(header.ID())
	if err != nil {
		return nil, err
	}
	txs := block.Transactions()
	bloomContent := &bloomContent{}
	for i, receipt := range receipts {
		bloomContent.add(receipt.GasPayer.Bytes())
		for _, output := range receipt.Outputs {
			for _, event := range output.Events {
				bloomContent.add(event.Address.Bytes())
				for _, topic := range event.Topics {
					bloomContent.add(topic.Bytes())
				}
			}
			for _, transfer := range output.Transfers {
				bloomContent.add(transfer.Sender.Bytes())
				bloomContent.add(transfer.Recipient.Bytes())
			}
		}
		origin, _ := txs[i].Origin()
		bloomContent.add(origin.Bytes())
	}
	signer, _ := header.Signer()
	bloomContent.add(signer.Bytes())
	bloomContent.add(header.Beneficiary().Bytes())

	k := bloom.LegacyEstimateBloomK(bloomContent.len())
	bloom := bloom.NewLegacyBloom(k)
	for _, item := range bloomContent.items {
		bloom.Add(item)
	}
	return &BeatMessage{
		Number:      header.Number(),
		ID:          header.ID(),
		ParentID:    header.ParentID(),
		Timestamp:   header.Timestamp(),
		TxsFeatures: uint32(header.TxsFeatures()),
		Bloom:       hexutil.Encode(bloom.Bits[:]),
		K:           uint32(k),
		Obsolete:    block.Obsolete,
	}, nil
}

type bloomContent struct {
	items [][]byte
}

func (bc *bloomContent) add(item []byte) {
	bc.items = append(bc.items, bytes.TrimLeft(item, "\x00"))
}

func (bc *bloomContent) len() int {
	return len(bc.items)
}
