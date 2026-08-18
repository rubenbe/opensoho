package records

// Simple function that converts the encryption field to a valid 6GHz field.
// Basically everything is sae except owe.
func EncryptionFor6GHz(encryption string) string {
	if encryption == "owe" {
		return "owe"
	}
	return "sae"
}
