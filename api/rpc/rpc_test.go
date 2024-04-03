package rpc

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/params"
	"log"
	"math/big"
	"testing"
)

var (
	signer = types.MakeSigner(params.MainnetChainConfig, big.NewInt(1))
)

func TestTransaction(t *testing.T) {
	tx2 := types.NewTransaction(0, common.HexToAddress("0xf077b491b355E64048cE21E3A6Fc4751eEeA77fa"), big.NewInt(1), 1000000, big.NewInt(100), nil)

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatal(err)
	}

	signingHash := signer.Hash(tx2)

	signature, err := secp256k1.Sign(signingHash.Bytes(), privateKey.D.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	r, s, v, err := signer.SignatureValues(tx2, signature)
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println(r, s, v)
	}

	tx, err := tx2.WithSignature(signer, signature)

	rawTx, _ := tx.MarshalJSON()
	hex := hexutil.Encode(rawTx)

	fmt.Println(hex)
}
