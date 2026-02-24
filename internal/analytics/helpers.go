package analytics

import "time"

// Helper functions for analytics package

func timePtr(t time.Time) *time.Time {
	return &t
}

func stringPtr(s string) *string {
	return &s
}
