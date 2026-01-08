package migration

import _ "embed"

//go:embed create-tables.sql
var Create string
