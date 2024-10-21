package app

import "time"

type Event struct {
	ID          int64
	Name        string
	Description string
	Timestamp   time.Time
	Source      string
	Severity    string
}
