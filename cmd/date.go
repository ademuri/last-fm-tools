package cmd

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

type ParsedDate struct {
	Date  time.Time
	Year  bool
	Month bool
	Day   bool
}

func parseDateRangeFromArgs(args []string) (start time.Time, end time.Time, err error) {
	switch len(args) {
	case 1:
		start, end, err = getImplicitDateRange(args[0])

	case 2:
		start, end, err = getExplicitDateRange(args[0], args[1])

	default:
		err = fmt.Errorf("Expected one or two date arguments")
	}
	return
}

func getImplicitDateRange(ds string) (start time.Time, end time.Time, err error) {
	date, err := parseSingleDatestring(ds)
	if err != nil {
		return
	}

	start = date.Date
	switch {
	case date.Year:
		end = start.AddDate(1, 0, 0)

	case date.Month:
		end = start.AddDate(0, 1, 0)

	case date.Day:
		end = start.AddDate(0, 0, 1)

	default:
		err = fmt.Errorf("Invalid format: %q", ds)
	}

	return
}

func getExplicitDateRange(startString, endString string) (start time.Time, end time.Time, err error) {
	startParsed, err := parseSingleDatestring(startString)
	if err != nil {
		return
	}
	start = startParsed.Date

	endParsed, err := parseSingleDatestring(endString)
	if err != nil {
		return
	}
	end = endParsed.Date

	return
}

func parseSingleDatestring(ds string) (date ParsedDate, err error) {
	matched, err := regexp.Match(`^\d{4}$`, []byte(ds))
	if err != nil {
		err = fmt.Errorf("Parsing datestring as year: %w", err)
		return
	}
	if matched {
		date.Date, err = time.Parse("2006", ds)
		if err != nil {
			err = fmt.Errorf("Parsing datestring as year: %w", err)
			return
		}
		date.Year = true
		return
	}

	matched, err = regexp.Match(`^\d{4}-\d{2}$`, []byte(ds))
	if err != nil {
		err = fmt.Errorf("Parsing datestring as month: %w", err)
		return
	}
	if matched {
		date.Date, err = time.Parse("2006-01", ds)
		if err != nil {
			err = fmt.Errorf("Parsing datestring as month: %w", err)
			return
		}
		date.Month = true
		return
	}

	matched, err = regexp.Match(`^\d{4}-\d{2}-\d{2}$`, []byte(ds))
	if err != nil {
		err = fmt.Errorf("Parsing datestring as day: %w", err)
		return
	}
	if matched {
		date.Date, err = time.Parse("2006-01-02", ds)
		if err != nil {
			err = fmt.Errorf("Parsing datestring as day: %w", err)
			return
		}
		date.Day = true
		return
	}

	// Try relative date format (e.g., 30d, 12w, 6m, 1y)
	// These are interpreted as "Ago" from now.
	reRelative := regexp.MustCompile(`^(\d+)([dwmy])$`)
	matches := reRelative.FindStringSubmatch(ds)
	if len(matches) == 3 {
		amount, err := strconv.Atoi(matches[1])
		if err != nil {
			return date, fmt.Errorf("parsing relative amount: %w", err)
		}
		
		unit := matches[2]
		now := time.Now()
		
		switch unit {
		case "d":
			date.Date = now.AddDate(0, 0, -amount)
		case "w":
			date.Date = now.AddDate(0, 0, -amount*7)
		case "m":
			date.Date = now.AddDate(0, -amount, 0)
		case "y":
			date.Date = now.AddDate(-amount, 0, 0)
		}
		
		// For relative dates, treating them as exact points in time (Day=true) is usually safe
		// for filtering logic which typically ignores the boolean flags and just uses .Date.
		date.Day = true 
		return date, nil
	}

	err = fmt.Errorf("Invalid format: %q", ds)
	return
}
