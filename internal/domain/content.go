package domain

import (
	"errors"
	"time"
)

// ContentBlock represents a dynamic content section (hero banners, announcements, etc.)
// L9: Supports scheduling with is_active, start_at, end_at for time-based campaigns.
type ContentBlock struct {
	ID         string     `json:"id"`
	SectionKey string     `json:"sectionKey"`
	Content    RawJSON    `json:"content"`
	IsActive   bool       `json:"isActive"`
	StartAt    *time.Time `json:"startAt,omitempty"`
	EndAt      *time.Time `json:"endAt,omitempty"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

// IsCurrentlyActive returns true if the content block is active and within its schedule.
// L9: Single point of truth for content visibility logic.
func (c *ContentBlock) IsCurrentlyActive() bool {
	if !c.IsActive {
		return false
	}
	now := time.Now()
	if c.StartAt != nil && now.Before(*c.StartAt) {
		return false
	}
	if c.EndAt != nil && now.After(*c.EndAt) {
		return false
	}
	return true
}

var (
	ErrContentNotFound = errors.New("content not found")
)
