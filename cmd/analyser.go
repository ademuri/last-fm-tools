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
	"bytes"
	"fmt"
	"time"

	"github.com/olekukonko/tablewriter"
)

type Analysis struct {
	results [][]string
	summary string
}

type Analyser interface {
	GetResults(dbPath string, user string, numToReturn int, start time.Time, end time.Time) (Analysis, error)

	GetName() string
}

func (a Analysis) String() string {
	out := new(bytes.Buffer)
	table := tablewriter.NewWriter(out)
	table.SetHeader(a.results[0])
	for _, row := range a.results[1:] {
		table.Append(row)
	}
	table.Render()
	fmt.Fprintf(out, "%s\n", a.summary)
	return out.String()
}
