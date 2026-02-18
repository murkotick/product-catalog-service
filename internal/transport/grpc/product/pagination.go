package product

import (
	"strconv"
)

// encodePageToken uses a simple, explicit offset string.
// This is good enough for the test task; you can switch to opaque tokens later.
func encodePageToken(offset int) string {
	if offset <= 0 {
		return ""
	}
	return strconv.Itoa(offset)
}

func decodePageToken(token string) (int, error) {
	if token == "" {
		return 0, nil
	}
	return strconv.Atoi(token)
}
