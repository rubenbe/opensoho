package records

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncryptionFor6GHz(t *testing.T) {
	nonOwe := []string{
		"sae",
		"sae-mixed",
		"psk2+tkip+ccmp",
		"psk2+tkip+aes",
		"psk2+tkip",
		"psk2+ccmp",
		"psk2+aes",
		"psk2",
		"psk-mixed+tkip+ccmp",
		"psk-mixed+tkip+aes",
		"psk-mixed+tkip",
		"psk-mixed+ccmp",
		"psk-mixed+aes",
		"psk-mixed",
		"psk+tkip+ccmp",
		"psk+tkip+aes",
		"psk+tkip",
		"psk+ccmp",
		"psk+aes",
		"psk",
		"none",
	}

	for _, encryption := range nonOwe {
		t.Run(encryption, func(t *testing.T) {
			assert.Equal(t, "sae", EncryptionFor6GHz(encryption))
		})
	}

	t.Run("owe", func(t *testing.T) {
		assert.Equal(t, "owe", EncryptionFor6GHz("owe"))
	})
}
