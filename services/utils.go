package services

import (
	"time"
	_ "time/tzdata"
)

// GetJakartaLocation returns *time.Location for Asia/Jakarta (WIB UTC+7),
// with fallback to FixedZone if not available.
func GetJakartaLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil || loc == nil {
		return time.FixedZone("WIB", 7*3600)
	}
	return loc
}

// GetToday returns today's date in YYYY-MM-DD format (Asia/Jakarta)
func GetToday() string {
	return today()
}
