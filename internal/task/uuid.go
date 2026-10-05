package task

import (
	"fmt"

	"github.com/google/uuid"
)

// NewUUID returns a canonical lowercase version 4 UUID.
func NewUUID() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("generate UUID: %w", err)
	}
	return id.String(), nil
}
