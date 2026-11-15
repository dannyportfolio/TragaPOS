package uid

import (
	"github.com/google/uuid"
)

func NewV7() string {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.New().String()
	}
	return id.String()
}

func IsValid(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}
