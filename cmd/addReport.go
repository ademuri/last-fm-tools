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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// addReportCmd represents the addReport command
var addReportCmd = &cobra.Command{
	Use:   "add-report <types...>",
	Short: "Adds an email report, to be sent periodically with `send-reports`",
	Long:  ``,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		params, _ := cmd.Flags().GetStringToString("params")
		err := addReport(viper.GetString("database"), viper.GetString("name"), viper.GetString("user"), viper.GetString("dest"), viper.GetInt("run_day"), args, params)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(addReportCmd)

	var email string
	addReportCmd.Flags().StringVar(&email, "dest", "", "Destination email address")
	addReportCmd.MarkFlagRequired("dest")
	err := viper.BindPFlag("dest", addReportCmd.Flags().Lookup("dest"))
	if err != nil {
		fmt.Println(err)
	}

	var reportName string
	addReportCmd.Flags().StringVar(&reportName, "name", "", "Report name - included in the email title, and used for periodically sending")
	addReportCmd.MarkFlagRequired("name")
	viper.BindPFlag("name", addReportCmd.Flags().Lookup("name"))

	var runDay int
	addReportCmd.Flags().IntVar(&runDay, "run_day", 0, "Which day of the month to run this report on")
	addReportCmd.MarkFlagRequired("runDay")
	viper.BindPFlag("run_day", addReportCmd.Flags().Lookup("run_day"))
	
	addReportCmd.Flags().StringToString("params", nil, "Parameters for reports (e.g. --params top-n=n=20)")
}

func addReport(dbPath string, name string, user string, to string, runDay int, types []string, params map[string]string) error {
	if runDay < 1 || runDay > 31 {
		return fmt.Errorf("run_day out of range: %d", runDay)
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
	
	// Convert params map to JSON
	// The input params is map[string]string where key is report name and value is params string.
	// We need to parse the params string into map[string]string for the storage if we want structured access, 
	// OR just store the raw string and parse later? 
	// Better to store structured: map[string]map[string]string
	
	structuredParams := make(map[string]map[string]string)
	for k, v := range params {
		// v is like "n=20,min=5" (comma separated? or custom format?)
		// StringToString uses comma to separate key=value pairs for the outer map.
		// So "top-n=n=20,min=5" -> key: "top-n", value: "n=20,min=5"
		// We'll treat the value as k=v pairs comma separated.
		pMap := make(map[string]string)
		pairs := strings.Split(v, ",")
		for _, pair := range pairs {
			kv := strings.SplitN(pair, "=", 2)
			if len(kv) == 2 {
				pMap[kv[0]] = kv[1]
			}
		}
		structuredParams[k] = pMap
	}

	paramsJSON, err := json.Marshal(structuredParams)
	if err != nil {
		return fmt.Errorf("marshalling params: %w", err)
	}

	_, err = db.Exec("INSERT INTO Report(user, name, email, run_day, types, params) VALUES (?, ?, ?, ?, ?, ?)", user, name, to, runDay, strings.Join(types, ","), string(paramsJSON))
	if err != nil {
		return err
	}

	return nil
}
