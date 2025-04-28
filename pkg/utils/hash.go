package utils

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
)

// CalculateSHA256 calculates the SHA-256 hash of data
func CalculateSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// CalculateSHA512 calculates the SHA-512 hash of data
func CalculateSHA512(data []byte) string {
	hash := sha512.Sum512(data)
	return hex.EncodeToString(hash[:])
}

// ByteSliceEqual compares two byte slices for equality
func ByteSliceEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
