package cmd

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strconv"

	"text/tabwriter"
	"time"

	"github.com/ademuri/last-fm-tools/internal/store"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var daysToCheck int

var checkSourcesCmd = &cobra.Command{
	Use:   "check-sources",
	Short: "Checks for gaps in scrobbling activity",
	Long:  `Analyzes scrobbles over a specified number of days (default 14) to detect potential failures in desktop (Work Hours) or mobile (Other Hours) scrobblers.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := checkSources(viper.GetString("database"), viper.GetString("user"), daysToCheck)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(checkSourcesCmd)
	checkSourcesCmd.Flags().IntVarP(&daysToCheck, "days", "D", 14, "Number of days to check back")
}

func checkSources(dbPath, user string, days int) error {
	analyzer := &CheckSourcesAnalyzer{}
	params := map[string]string{
		"days": strconv.Itoa(days),
	}
	if err := analyzer.Configure(params); err != nil {
		return err
	}

	// Use dummy times for GetResults as it calculates its own window based on 'days'
	res, err := analyzer.GetResults(dbPath, user, time.Time{}, time.Time{})
	if err == ErrSkipReport {
		fmt.Println("No scrobbling issues detected.")
		return nil
	}
	if err != nil {
		return err
	}

	fmt.Println(res.BodyOverride)
	return nil
}

type CheckSourcesAnalyzer struct {
	Days     int
	Timezone string
}

func (c *CheckSourcesAnalyzer) GetName() string {
	return "Scrobble Check"
}

func (c *CheckSourcesAnalyzer) Configure(params map[string]string) error {
	if val, ok := params["days"]; ok {
		d, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid value for 'days': %v", err)
		}
		c.Days = d
	} else {
		c.Days = 14 // Default
	}

	if val, ok := params["timezone"]; ok {
		c.Timezone = val
	}
	return nil
}

func (c *CheckSourcesAnalyzer) GetResults(dbPath string, user string, _ time.Time, _ time.Time) (Analysis, error) {
	db, err := store.New(dbPath)
	if err != nil {
		return Analysis{}, fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	// Resolve Location
	loc := time.Local
	if c.Timezone != "" {
		l, err := time.LoadLocation(c.Timezone)
		if err != nil {
			return Analysis{}, fmt.Errorf("loading timezone %q: %w", c.Timezone, err)
		}
		loc = l
	}

	// Analyze last N days
	now := time.Now().In(loc)
	// Start from N days ago at 00:00:00 local time
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	start := todayStart.AddDate(0, 0, -c.Days)
	end := now

	listens, err := db.GetListensInRange(user, start, end)
	if err != nil {
		return Analysis{}, fmt.Errorf("getting listens: %w", err)
	}

	// Buckets:
	// Day -> { WorkHours, OtherHours }
	type DayCounts struct {
		Date       time.Time
		WorkHours  int // Mon-Fri 09:00 - 17:00
		OtherHours int
	}

	counts := make(map[string]*DayCounts)

	// Initialize counts for all days in range
	for d := start; d.Before(end) || d.Equal(end); d = d.AddDate(0, 0, 1) {
		if d.After(now) {
			break
		}
		dateStr := d.Format("2006-01-02")
		counts[dateStr] = &DayCounts{Date: d}
	}

	for _, t := range listens {
		// Ensure t is in target location
		tLocal := t.In(loc)

		dateStr := tLocal.Format("2006-01-02")

		if _, ok := counts[dateStr]; !ok {
			counts[dateStr] = &DayCounts{Date: tLocal}
		}

		isWeekend := tLocal.Weekday() == time.Saturday || tLocal.Weekday() == time.Sunday
		hour := tLocal.Hour()
		isWorkHour := hour >= 9 && hour < 17 // 9:00 - 16:59

		if !isWeekend && isWorkHour {
			counts[dateStr].WorkHours++
		} else {
			counts[dateStr].OtherHours++
		}
	}

	// Prepare data for table
	var keys []string
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

		// Calculate streaks
		workStreak := 0
		otherStreak := 0
		weekendStreak := 0
	
		// Other Streak
		for i := len(keys) - 1; i >= 0; i-- {
			if counts[keys[i]].OtherHours == 0 {
				otherStreak++
			} else {
				break
			}
		}
	
		// Work Streak (skip weekends)
		for i := len(keys) - 1; i >= 0; i-- {
			entry := counts[keys[i]]
			isWeekend := entry.Date.Weekday() == time.Saturday || entry.Date.Weekday() == time.Sunday
			if isWeekend {
				continue
			}
			if entry.WorkHours == 0 {
				workStreak++
			} else {
				break
			}
		}
	
		// Weekend Streak (skip weekdays)
		for i := len(keys) - 1; i >= 0; i-- {
			entry := counts[keys[i]]
			isWeekend := entry.Date.Weekday() == time.Saturday || entry.Date.Weekday() == time.Sunday
			if !isWeekend {
				continue
			}
			// On weekends, all hours are "OtherHours" (WorkHours is 0 by definition in our loop)
			// But let's check total listens just to be safe (OtherHours + WorkHours)
			if entry.OtherHours+entry.WorkHours == 0 {
				weekendStreak++
			} else {
				break
			}
		}
	
		if workStreak <= 3 && otherStreak <= 3 && weekendStreak < 4 {
			return Analysis{}, ErrSkipReport
		}
	
		// Generate Output
		var buf bytes.Buffer
		
		fmt.Fprintf(&buf, "Scrobble Check for user: %s (Timezone: %s)\n", user, loc.String())
		fmt.Fprintln(&buf, "Work Hours: Mon-Fri, 09:00 - 17:00")
		fmt.Fprintln(&buf)
	
		if workStreak > 3 {
			fmt.Fprintf(&buf, "⚠️  Potential Work Scrobbler Failure: No listens during work hours for the last %d working days.\n", workStreak)
		}
		if weekendStreak >= 4 {
			fmt.Fprintf(&buf, "⚠️  Potential Weekend Scrobbler Failure: No listens during weekends for the last %d weekend days.\n", weekendStreak)
		}
		if otherStreak > 3 {
			fmt.Fprintf(&buf, "⚠️  Potential Mobile/Home Scrobbler Failure: No listens during off-hours for the last %d days.\n", otherStreak)
		}
		fmt.Fprintln(&buf)
		w := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "Date\tDay\tWork Hours (9-5)\tOther Hours")

	for _, dateStr := range keys {
		c := counts[dateStr]
		dayName := c.Date.Weekday().String()[:3]
		workStr := fmt.Sprintf("%d", c.WorkHours)
		otherStr := fmt.Sprintf("%d", c.OtherHours)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", dateStr, dayName, workStr, otherStr)
	}
	w.Flush()

	return Analysis{
		BodyOverride: buf.String(),
		summary:      "Scrobbling issues detected",
	}, nil
}
