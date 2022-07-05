// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package atlas

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// CustomIso8601Format is an addition format that we see generated in JSON from
// the Java side of the house.
const CustomIso8601Format = "2006-01-02T15:04Z"

// timeFormats containst the order and formats accepted by the JSON marshaler
var timeFormats = []string{
	time.RFC3339,
	CustomIso8601Format,
}

// Time is an alias for the standard library time. It exists because we need a custom
// JSON unmarshal step that accepts ISO 8601 instead of just RFC3339.
type Time time.Time

// Now gets the current time in UTC.
func Now() Time {
	return Time(time.Now().UTC())
}

// UnmarshalJSON parses a Time from JSON. It attempts to use the default RFC3339 format.
// On failure, it uses the custom time format.
func (t *Time) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	return t.ParseTime(s)
}

// Parse a string to time
func (t *Time) ParseTime(s string) error {
	for _, format := range timeFormats {
		parsed, err := time.Parse(format, s)
		if err == nil {
			*t = Time(parsed)
			return nil
		}
	}
	return fmt.Errorf("unmarshal time: invalid format")
}

// MarshalJSON converts this timestamp to JSON.
func (t Time) MarshalJSON() ([]byte, error) {
	return time.Time(t).MarshalJSON()
}

// String converts this timestamp to a string.
func (t Time) String() string {
	return time.Time(t).String()
}

// Return the customISO string for formatting upstream
func CreateIsoFormattedTime(rfc3339FormattedTime string) (Time, error) {
	parsed, err := time.Parse(CustomIso8601Format, rfc3339FormattedTime)
	return Time(parsed), err
}

func (leftTime Time) After(rightTime Time) bool {
	return time.Time(leftTime).After(time.Time(rightTime))
}

func (leftTime Time) Before(rightTime Time) bool {
	return time.Time(leftTime).Before(time.Time(rightTime))
}

func (t Time) Year() int {
	return time.Time(t).Year()
}

// SleepWithContext will sleep the current go routine for the specified
// duration. The method will return early if the context is cancelled.
func SleepWithContext(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
