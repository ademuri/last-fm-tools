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
	"database/sql"
	"fmt"
	"strconv"
	"time"
)

func createListenForDate(db *sql.DB, user string, time time.Time) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("creating transaction: %w", err)
	}
	err = createListen(tx, user, 1, strconv.FormatInt(time.Unix(), 10))
	if err != nil {
		return fmt.Errorf("createListen(): %w", err)
	}
	tx.Commit()

	return nil
}