package e2e

import (
	"fmt"
)

var (
	node1 = &clientEndpoints{"http://localhost:8669"}

	node2 = &clientEndpoints{"http://localhost:8679"}
	node3 = &clientEndpoints{"http://localhost:8689"}
)

type clientEndpoints struct {
	baseUrl string
}

func (c *clientEndpoints) getAccount(address string) string {
	return fmt.Sprintf("%s/accounts/%s", c.baseUrl, address)
}

func (c *clientEndpoints) getExpandedBlock(revision string) string {
	return fmt.Sprintf("%s/blocks/%s?expanded=true", c.baseUrl, revision)
}

func (c *clientEndpoints) getCompressedBlock(revision string) string {
	return fmt.Sprintf("%s/blocks/%s", c.baseUrl, revision)
}

func (c *clientEndpoints) postTransaction() string {
	return fmt.Sprintf("%s/transactions", c.baseUrl)
}

func (c *clientEndpoints) getTransaction(id string) string {
	return fmt.Sprintf("%s/transactions/%s", c.baseUrl, id)
}

func (c *clientEndpoints) getTransactionReceipt(id string) string {
	return fmt.Sprintf("%s/transactions/%s/receipt", c.baseUrl, id)
}
