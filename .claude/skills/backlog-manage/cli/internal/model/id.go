package model

import (
	"fmt"
	"math/rand/v2"
	"time"
)

// NowFunc is a function that returns the current time. Override in tests.
var NowFunc = func() time.Time { return time.Now().UTC() }

func nowUTC() string {
	return NowFunc().Format(time.RFC3339)
}

// GenerateID creates an ID in the format {type}-{YYYYMMDD}-{4hex}.
func GenerateID(itemType string) string {
	now := NowFunc()
	hex := rand.IntN(65536)
	return fmt.Sprintf("%s-%s-%04x", itemType, now.Format("20060102"), hex)
}
