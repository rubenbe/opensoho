package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTzData(t *testing.T) {
	assert.Equal(t, "non-existing", GetTzData("non-existing"))
	assert.Equal(t, "", GetTzData("Disabled"))
	assert.Equal(t, "", GetTzData("UTC"))
	assert.Equal(t, "CET-1CEST,M3.5.0,M10.5.0/3", GetTzData("Europe/Brussels"))
	assert.Equal(t, "EST5EDT,M3.2.0,M11.1.0", GetTzData("America/New_York"))
}
