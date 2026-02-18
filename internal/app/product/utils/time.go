package utils

import "time"

// ParseTimePtr parses an RFC3339 string pointer into *time.Time.
// Returns nil if input is nil, empty, or parsing fails.
func ParseTimePtr(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil
	}
	tt := t.UTC()
	return &tt
}

// TimeOrZero returns the dereferenced time or zero time if nil.
func TimeOrZero(p *time.Time) time.Time {
	if p == nil {
		return time.Time{}
	}
	return *p
}
