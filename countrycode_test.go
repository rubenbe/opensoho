package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidCountryCode(t *testing.T) {
	assert.Equal(t, true, IsValidCountryCode(""))
	assert.Equal(t, true, IsValidCountryCode("00"))
	assert.Equal(t, true, IsValidCountryCode("BE"))
	assert.Equal(t, true, IsValidCountryCode("US"))

	assert.Equal(t, false, IsValidCountryCode("ZZZ"))
	assert.Equal(t, false, IsValidCountryCode("be"))
}
