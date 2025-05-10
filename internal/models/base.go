package models

import (
	"strconv" // Import strconv for uint to string conversion
	"time"

	"gorm.io/gorm"
)

// BaseModel defines the common fields for all models.
// It includes an auto-incrementing ID, and CreatedAt and UpdatedAt timestamps.
type BaseModel struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"` // For soft deletes
}

// IDString returns the ID as a string.
func (b *BaseModel) IDString() string {
	return strconv.FormatUint(uint64(b.ID), 10)
}
