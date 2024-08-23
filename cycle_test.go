package main

import (
	"testing"
	"time"
)

func TestParseCycleIn(t *testing.T) {
	tests := []struct {
		cycle   string
		in      string
		want    time.Time
		wantErr bool
	}{
		{"hourly", "30", time.Now().Truncate(time.Hour).Add(30 * time.Minute), false},
		{"daily", "noon", time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 12, 0, 0, 0, time.Local), false},
		{"daily", "midnight", time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Local), false},
		{"weekly", "monday", time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 3, 0, 0, 0, time.Local).AddDate(0, 0, (7+int(time.Monday-time.Now().Weekday()))%7), false},
		{"monthly", "15", time.Date(time.Now().Year(), time.Now().Month(), 15, 3, 0, 0, 0, time.Local), false},
		{"annually", "12-25", time.Date(time.Now().Year(), time.December, 25, 3, 0, 0, 0, time.Local), false},
		{"hourly", "invalid", time.Time{}, true},
		{"daily", "25", time.Time{}, true},
		{"weekly", "funday", time.Time{}, true},
		{"monthly", "45", time.Time{}, true},
		{"annually", "13-32", time.Time{}, true},
	}

	now := time.Now()
	for _, tt := range tests {
		t.Run(tt.cycle+" "+tt.in, func(t *testing.T) {
			got, err := parseCycleIn(tt.cycle, tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCycleIn() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Adjust for cycles that depend on the current date
			if tt.cycle == "hourly" {
				expected := now.Truncate(time.Hour).Add(30 * time.Minute)
				if now.After(expected) {
					expected = expected.Add(time.Hour)
				}
				tt.want = expected
			}

			if tt.cycle == "daily" && (tt.in == "noon" || tt.in == "midnight") {
				hour, min := 12, 0
				if tt.in == "midnight" {
					hour, min = 0, 0
				}
				expected := time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, time.Local)
				if now.After(expected) {
					expected = expected.Add(24 * time.Hour)
				}
				tt.want = expected
			}

			if tt.cycle == "monthly" {
				expected := time.Date(now.Year(), now.Month(), 15, 3, 0, 0, 0, time.Local)
				if now.Day() > 15 {
					expected = expected.AddDate(0, 1, 0)
				}
				tt.want = expected
			}

			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("parseCycleIn() = %v, want %v", got, tt.want)
			}
		})
	}
}
