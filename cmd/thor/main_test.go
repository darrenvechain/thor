package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoNetwork(t *testing.T) {
	err := run(make([]string, 1))
	assert.Equal(t, err.Error(), "network flag not specified")
}
