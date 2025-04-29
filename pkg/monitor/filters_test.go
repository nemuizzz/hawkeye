package monitor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegexFilter(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		replacement string
		input       string
		expected    string
	}{
		{
			name:        "simple replacement",
			pattern:     "test",
			replacement: "replaced",
			input:       "this is a test string",
			expected:    "this is a replaced string",
		},
		{
			name:        "regex with capture groups",
			pattern:     `(\d+)`,
			replacement: "NUM",
			input:       "value: 12345 and another: 6789",
			expected:    "value: NUM and another: NUM",
		},
		{
			name:        "no match",
			pattern:     "xyz",
			replacement: "abc",
			input:       "this has no match",
			expected:    "this has no match",
		},
		{
			name:        "empty input",
			pattern:     "test",
			replacement: "replaced",
			input:       "",
			expected:    "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			filter, err := NewRegexFilter(tc.pattern, tc.replacement, "Test filter")
			require.NoError(t, err)
			require.NotNil(t, filter)

			result := filter.Apply([]byte(tc.input))
			require.Equal(t, tc.expected, string(result))
		})
	}
}

func TestTimestampFilter(t *testing.T) {
	filter, err := NewTimestampFilter()
	require.NoError(t, err)
	require.NotNil(t, filter)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "ISO8601 timestamp",
			input:    "Last updated: 2023-04-15T14:32:17Z",
			expected: "Last updated: TIMESTAMP",
		},
		{
			name:     "timestamp with timezone offset",
			input:    "Published: 2023-04-15T14:32:17+09:00",
			expected: "Published: TIMESTAMP",
		},
		{
			name:     "compact timestamp",
			input:    "Generated: 202304150212+0900",
			expected: "Generated: TIMESTAMP",
		},
		{
			name:     "unix timestamp",
			input:    "Timestamp: 1681543937",
			expected: "Timestamp: TIMESTAMP",
		},
		{
			name:     "millisecond timestamp",
			input:    "MS Timestamp: 1681543937123",
			expected: "MS Timestamp: TIMESTAMP",
		},
		{
			name:     "no timestamp",
			input:    "This text has no timestamp",
			expected: "This text has no timestamp",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := filter.Apply([]byte(tc.input))
			require.Equal(t, tc.expected, string(result))
		})
	}
}

func TestDateFilter(t *testing.T) {
	filter, err := NewDateFilter()
	require.NoError(t, err)
	require.NotNil(t, filter)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "ISO format date",
			input:    "Date: 2023-04-15",
			expected: "Date: DATE",
		},
		{
			name:     "US format date",
			input:    "Published on 04/15/2023",
			expected: "Published on DATE",
		},
		{
			name:     "EU format date",
			input:    "Updated on 15.04.2023",
			expected: "Updated on DATE",
		},
		{
			name:     "multiple dates",
			input:    "From 2023-01-01 to 2023-12-31",
			expected: "From DATE to DATE",
		},
		{
			name:     "no date",
			input:    "This text has no date",
			expected: "This text has no date",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := filter.Apply([]byte(tc.input))
			require.Equal(t, tc.expected, string(result))
		})
	}
}

func TestContentFilterList(t *testing.T) {
	// Create multiple filters
	dateFilter, err := NewDateFilter()
	require.NoError(t, err)

	tsFilter, err := NewTimestampFilter()
	require.NoError(t, err)

	wordFilter, err := NewRegexFilter("sensitive", "REDACTED", "Redact sensitive words")
	require.NoError(t, err)

	// Create filter list
	filterList := ContentFilterList{tsFilter, dateFilter, wordFilter}

	// Test all filters applied in sequence
	input := "Created on 2023-04-15T14:32:17Z with sensitive data"
	expected := "Created on TIMESTAMP with REDACTED data"

	result := filterList.Apply([]byte(input))
	require.Equal(t, expected, string(result))
}

func TestCreateDefaultFilters(t *testing.T) {
	filters, err := CreateDefaultFilters()
	require.NoError(t, err)
	require.Len(t, filters, 2, "Should have created 2 default filters")

	// Test the filters
	input := "Date: 2023-04-15, Timestamp: 2023-04-15T14:32:17Z"
	expected := "Date: DATE, Timestamp: TIMESTAMP"

	result := filters.Apply([]byte(input))
	require.Equal(t, expected, string(result))
}
