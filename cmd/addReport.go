/*
Copyright 2020 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// addReportCmd represents the addReport command
var addReportCmd = &cobra.Command{
	Use:   "add-report <types...>",
	Short: "Adds an email report, to be sent periodically with `send-reports`",
	Long:  ``,
	Args:  cobra.MinimumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		dest, _ := cmd.Flags().GetString("dest")
		if dest == "" {
			return fmt.Errorf("required flag(s) \"dest\" not set")
		}

		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			return fmt.Errorf("required flag(s) \"name\" not set")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		params, _ := cmd.Flags().GetStringArray("params")

		if len(params) > 0 && len(params) != len(args) {
			fmt.Printf("Error: Number of --params flags (%d) must match number of reports (%d), or be 0.\n", len(params), len(args))
			os.Exit(1)
		}

		dest, _ := cmd.Flags().GetString("dest")
		name, _ := cmd.Flags().GetString("name")
		runDay, _ := cmd.Flags().GetInt("run_day")
		interval, _ := cmd.Flags().GetInt("interval")
		firstRun, _ := cmd.Flags().GetString("first_run")

		err := addReport(viper.GetString("database"), name, viper.GetString("user"), dest, runDay, interval, firstRun, args, params)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(addReportCmd)

	addReportCmd.Flags().String("dest", "", "Destination email address")
	addReportCmd.Flags().String("name", "", "Report name - included in the email title, and used for periodically sending")
	addReportCmd.Flags().Int("run_day", 0, "Which day of the month to run this report on")
	addReportCmd.Flags().Int("interval", 0, "Interval in days between reports")
	addReportCmd.Flags().String("first_run", "", "Date of the first run (YYYY-MM-DD)")
	addReportCmd.Flags().StringArray("params", nil, "Parameters for reports, matched by index (e.g. --params 'n=20')")
}

func addReport(dbPath string, name string, user string, to string, runDay int, interval int, firstRun string, types []string, params []string) error {
	if runDay < 0 || runDay > 31 {
		return fmt.Errorf("run_day out of range: %d", runDay)
	}

	var nextRunTime time.Time
	if firstRun != "" {
		var err error
		nextRunTime, err = time.Parse("2006-01-02", firstRun)
		if err != nil {
			return fmt.Errorf("invalid first_run date: %w", err)
		}
	} else if interval > 0 {
		nextRunTime = time.Now()
	}

	for _, actionName := range types {
		_, err := getActionFromName(actionName)
		if err != nil {
			return fmt.Errorf("Invalid type: %q", actionName)
		}
	}

	if len(to) == 0 {
		return fmt.Errorf("Must specify destination email")
	}

	db, err := createDatabase(dbPath)
	if err != nil {
		return err
	}
	err = createUser(db, user)
	if err != nil {
		return nil
	}

	// Convert params slice to JSON list of maps
	structuredParams := make([]map[string]string, len(types))
	for i := range types {
		pMap := make(map[string]string)
		if i < len(params) && params[i] != "" {
			pairs := strings.Split(params[i], ",")
			for _, pair := range pairs {
				kv := strings.SplitN(pair, "=", 2)
				if len(kv) == 2 {
					pMap[kv[0]] = kv[1]
				}
			}
		}
		structuredParams[i] = pMap
	}

	paramsJSON, err := json.Marshal(structuredParams)
	if err != nil {
		return fmt.Errorf("marshalling params: %w", err)
	}

	var nextRun interface{}
	if !nextRunTime.IsZero() {
		nextRun = nextRunTime
	}

	_, err = db.Exec("INSERT INTO Report(user, name, email, run_day, types, params, interval_days, next_run) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", user, name, to, runDay, strings.Join(types, ","), string(paramsJSON), interval, nextRun)
	if err != nil {
		return err
	}

	return nil
}
