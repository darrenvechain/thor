package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/vechain/thor/v2/api/accounts"
	"github.com/vechain/thor/v2/api/blocks"
	"github.com/vechain/thor/v2/api/transactions"
	"io"
	"net/http"
)

func getAccount(url string) (*accounts.Account, error) {
	return httpGet(url, new(accounts.Account))
}

func getExpandedBlock(url string) (*blocks.JSONExpandedBlock, error) {
	return httpGet(url, new(blocks.JSONExpandedBlock))
}

func getTransactionReceipt(url string) (*transactions.Receipt, error) {
	return httpGet(url, new(transactions.Receipt))
}

func getCompressedBlock(url string) (*blocks.JSONCollapsedBlock, error) {
	return httpGet(url, new(blocks.JSONCollapsedBlock))
}

func getTransaction(url string) (*transactions.Transaction, error) {
	return httpGet(url, new(transactions.Transaction))
}

func postTransaction(url string, body transactions.RawTx) (*transactions.SubmitTxResponse, error) {
	return httpPost(url, body, new(transactions.SubmitTxResponse))
}

// HttpGet sends an HTTP GET request to the specified URL.
// It then decodes the response body into the provided response type.
func httpGet[T any](url string, v *T) (*T, error) {
	return httpRequest(http.MethodGet, url, nil, v)
}

// HttpPost sends an HTTP POST request to the specified URL with the given request body.
// It then decodes the response body into the provided response type.
func httpPost[T any](url string, requestBody interface{}, v *T) (*T, error) {
	return httpRequest(http.MethodPost, url, requestBody, v)
}

// httpError is a custom error type for HTTP errors
type httpError struct {
	Message string
	Code    int
}

// Error implements the error interface for httpError
func (he *httpError) Error() string {
	return fmt.Sprintf("HTTP error: %s (Code: %d)", he.Message, he.Code)
}

// HttpRequest sends an HTTP request with the specified method to the given URL.
// It then decodes the response body into the provided response type.
func httpRequest[T any](method, url string, requestBody interface{}, v *T) (*T, error) {
	// Encode request body to JSON
	var requestBodyJSON []byte
	if requestBody != nil {
		var err error
		requestBodyJSON, err = json.Marshal(requestBody)
		if err != nil {
			return nil, fmt.Errorf("failed to encode request body: %v", err)
		}
	}

	// Create request based on the specified method
	req, err := http.NewRequest(method, url, bytes.NewBuffer(requestBodyJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Perform the HTTP request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Check if the response status code indicates an error
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Read the error message from the response body
		errorBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read error response body: %v", err)
		}

		// Return a custom httpError
		return nil, &httpError{
			Message: string(errorBody),
			Code:    resp.StatusCode,
		}
	}

	// Read the response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// Check if the response body is "null"
	if string(responseBody) == "null\n" {
		return nil, &httpError{
			Message: "HTTP response body is null",
			Code:    resp.StatusCode,
		}
	}

	// Read the response body
	err = json.Unmarshal(responseBody, &v)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response body: %v", err)
	}

	return v, nil
}
