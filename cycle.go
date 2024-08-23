package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func parseCycleIn(cycle string, cycleIn string) (time.Time, error) {
	now := time.Now()
	location := now.Location()

	switch strings.ToLower(cycle) {
	case "hourly":
		minutes, err := strconv.Atoi(cycleIn)
		if err != nil || minutes < 0 || minutes >= 60 {
			return time.Time{}, fmt.Errorf("invalid minutes for hourly cycle: %w", err)
		}
		nextHour := now.Truncate(time.Hour).Add(time.Duration(minutes) * time.Minute)
		if nextHour.Before(now) {
			nextHour = nextHour.Add(time.Hour)
		}
		return nextHour, nil

	case "daily":
		var hourMinute time.Time
		switch cycleIn {
		case "noon":
			hourMinute = time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, location)
		case "midnight":
			hourMinute = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
		default:
			parsed, err := time.Parse("15:04", cycleIn)
			if err != nil {
				return time.Time{}, fmt.Errorf("invalid time format for daily cycle: %w", err)
			}
			hourMinute = time.Date(now.Year(), now.Month(), now.Day(), parsed.Hour(), parsed.Minute(), 0, 0, location)
		}
		if hourMinute.Before(now) {
			hourMinute = hourMinute.AddDate(0, 0, 1)
		}
		return hourMinute, nil

	case "weekly":
		dayOfWeek := strings.ToLower(cycleIn)
		targetWeekday, err := parseWeekday(dayOfWeek)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid day for weekly cycle: %w", err)
		}
		nextWeek := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, location)
		for nextWeek.Weekday() != targetWeekday {
			nextWeek = nextWeek.AddDate(0, 0, 1)
		}
		if nextWeek.Before(now) {
			nextWeek = nextWeek.AddDate(0, 0, 7)
		}
		return nextWeek, nil

	case "monthly":
		dayOfMonth, err := strconv.Atoi(cycleIn)
		if err != nil || dayOfMonth < 1 || dayOfMonth > 31 {
			return time.Time{}, fmt.Errorf("invalid day for monthly cycle: %w", err)
		}
		nextMonth := time.Date(now.Year(), now.Month(), dayOfMonth, 3, 0, 0, 0, location)
		if nextMonth.Before(now) {
			nextMonth = nextMonth.AddDate(0, 1, 0)
		}
		return nextMonth, nil

	case "annually":
		monthDay := strings.Split(cycleIn, "-")
		if len(monthDay) != 2 {
			return time.Time{}, fmt.Errorf("invalid format for annual cycle, expected MM-DD: %s", cycleIn)
		}
		month, err := strconv.Atoi(monthDay[0])
		if err != nil || month < 1 || month > 12 {
			return time.Time{}, fmt.Errorf("invalid month for annual cycle: %s", cycleIn)
		}
		day, err := strconv.Atoi(monthDay[1])
		if err != nil || day < 1 || day > 31 {
			return time.Time{}, fmt.Errorf("invalid day for annual cycle: %s", cycleIn)
		}
		nextYear := time.Date(now.Year(), time.Month(month), day, 3, 0, 0, 0, location)
		if nextYear.Before(now) {
			nextYear = nextYear.AddDate(1, 0, 0)
		}
		return nextYear, nil

	default:
		return time.Time{}, fmt.Errorf("unknown cycle: %s", cycle)
	}
}

func parseWeekday(day string) (time.Weekday, error) {
	switch day {
	case "sunday":
		return time.Sunday, nil
	case "monday":
		return time.Monday, nil
	case "tuesday":
		return time.Tuesday, nil
	case "wednesday":
		return time.Wednesday, nil
	case "thursday":
		return time.Thursday, nil
	case "friday":
		return time.Friday, nil
	case "saturday":
		return time.Saturday, nil
	default:
		return time.Sunday, fmt.Errorf("invalid weekday: %s", day)
	}
}
