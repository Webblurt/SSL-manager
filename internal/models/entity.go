package models

import "time"

type Entity struct {
	EntityName        string // must match the table name
	StringParameters  map[string]string
	IntegerParameters map[string]int
	TimeParameters    map[string]time.Time
	BoolParameters    map[string]bool
}
