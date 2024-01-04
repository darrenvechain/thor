package endpoints

import (
	"fmt"
)

var (
	Node1 = &ClientEndpoints{"http://127.0.0.1:8669"}

	Node2 = &ClientEndpoints{"http://127.0.0.1:8679"}
	Node3 = &ClientEndpoints{"http://127.0.0.1:8689"}
)

type ClientEndpoints struct {
	baseUrl string
}

func (c *ClientEndpoints) GetAccount(address string) string {
	return fmt.Sprintf("%s/accounts/%s", c.baseUrl, address)
}

func (c *ClientEndpoints) GetExpandedBlock(number int32) string {
	return fmt.Sprintf("%s/blocks/%d?expanded=true", c.baseUrl, number)
}

func (c *ClientEndpoints) GetCompressedBlock(number int32) string {
	return fmt.Sprintf("%s/blocks/%d", c.baseUrl, number)
}
