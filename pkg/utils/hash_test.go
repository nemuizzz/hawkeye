package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalculateSHA256(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "empty string",
			input:    []byte(""),
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "hello world",
			input:    []byte("hello world"),
			expected: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateSHA256(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestByteSliceEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        []byte
		b        []byte
		expected bool
	}{
		{
			name:     "empty slices",
			a:        []byte{},
			b:        []byte{},
			expected: true,
		},
		{
			name:     "equal slices",
			a:        []byte{1, 2, 3},
			b:        []byte{1, 2, 3},
			expected: true,
		},
		{
			name:     "different lengths",
			a:        []byte{1, 2, 3},
			b:        []byte{1, 2},
			expected: false,
		},
		{
			name:     "different content",
			a:        []byte{1, 2, 3},
			b:        []byte{1, 2, 4},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ByteSliceEqual(tt.a, tt.b)
			require.Equal(t, tt.expected, result)
		})
	}
}
