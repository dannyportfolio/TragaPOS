package uid

import (
	"github.com/google/uuid"
)

// NewV7 generates a new time-ordered UUIDv7 string.
// UUIDv7 embeds a millisecond-precision Unix timestamp in the most significant 48 bits,
// guaranteeing natural temporal ordering while maintaining global collision-free uniqueness
// across independent distributed edge nodes without coordination.
func NewV7() string {
	id, err := uuid.NewV7()
	if err != nil {
		// Fallback to V4 in the extremely unlikely event of entropy failure
		return uuid.NewString()
	}
	return id.String()
}

// IsValid checks if a given string is a valid UUID representation.
func IsValid(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
