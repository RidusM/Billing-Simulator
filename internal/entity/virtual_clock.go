package entity

import "time"

type VirtualClock struct {
	ID            int
	CurrentTime   time.Time
	OffsetSeconds int64
	UpdatedAt     time.Time
	UpdatedBy     string
}
