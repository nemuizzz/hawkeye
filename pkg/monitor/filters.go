package monitor

import (
	"regexp"
)

// ContentFilter defines an interface for filtering content before comparison
type ContentFilter interface {
	// Apply filters the content and returns the filtered version
	Apply(content []byte) []byte
	// Description returns a human-readable description of the filter
	Description() string
}

// RegexFilter is a filter that removes or replaces content matching a regex pattern
type RegexFilter struct {
	pattern     *regexp.Regexp
	replacement []byte
	description string
}

// NewRegexFilter creates a new regex-based content filter
func NewRegexFilter(pattern string, replacement string, description string) (*RegexFilter, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	return &RegexFilter{
		pattern:     re,
		replacement: []byte(replacement),
		description: description,
	}, nil
}

// Apply implements ContentFilter.Apply
func (f *RegexFilter) Apply(content []byte) []byte {
	return f.pattern.ReplaceAll(content, f.replacement)
}

// Description implements ContentFilter.Description
func (f *RegexFilter) Description() string {
	return f.description
}

// TimestampFilter is a specialized filter for ignoring common timestamp formats
func NewTimestampFilter() (*RegexFilter, error) {
	// This pattern matches common timestamp formats
	// ISO8601, RFC3339, unix timestamps, and more basic formats
	pattern := `(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}([+-]\d{2}:?\d{2}|Z))|` +
		`(\d{4}\d{2}\d{2}\d{2}\d{2}[+-]\d{4})|` +
		`(\d{10,13})` // Unix timestamp (seconds or milliseconds)

	return NewRegexFilter(pattern, "TIMESTAMP", "Ignore timestamps")
}

// DateFilter ignores common date formats
func NewDateFilter() (*RegexFilter, error) {
	pattern := `\d{4}-\d{2}-\d{2}|\d{2}/\d{2}/\d{4}|\d{2}\.\d{2}\.\d{4}`
	return NewRegexFilter(pattern, "DATE", "Ignore date strings")
}

// ContentFilterList is a collection of content filters to be applied in sequence
type ContentFilterList []ContentFilter

// Apply runs all filters in the list
func (l ContentFilterList) Apply(content []byte) []byte {
	result := content
	for _, filter := range l {
		result = filter.Apply(result)
	}
	return result
}

// CreateDefaultFilters returns a standard set of filters
func CreateDefaultFilters() (ContentFilterList, error) {
	var filters ContentFilterList

	// Add timestamp filter
	tsFilter, err := NewTimestampFilter()
	if err != nil {
		return nil, err
	}
	filters = append(filters, tsFilter)

	// Add date filter
	dateFilter, err := NewDateFilter()
	if err != nil {
		return nil, err
	}
	filters = append(filters, dateFilter)

	return filters, nil
}
