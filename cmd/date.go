package cmd

import (
	"fmt"
	"regexp"
	"time"
)

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

	err = fmt.Errorf("Invalid format: %q", ds)
	return
}
