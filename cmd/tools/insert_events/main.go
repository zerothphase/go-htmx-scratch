package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/zerothphase/go-htmx-scratch/internal/app"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <number_of_events>")
		os.Exit(1)
	}

	numEvents, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatal("Invalid number of events:", err)
	}

	db, err = sql.Open("sqlite3", "./events.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	for i := 0; i < numEvents; i++ {
		event := generateRandomEvent()
		err := insertEvent(event)
		if err != nil {
			log.Printf("Error inserting event: %v", err)
		} else {
			fmt.Printf("Inserted event: %s\n", event.Name)
		}
	}

	fmt.Printf("Inserted %d events\n", numEvents)
}

func generateRandomEvent() app.Event {
	return app.Event{
		Name:        fmt.Sprintf("Event %d", rand.Intn(1000)),
		Description: fmt.Sprintf("This is a random event description %d", rand.Intn(1000)),
		Timestamp:   time.Now().Add(-time.Duration(rand.Intn(7*24)) * time.Hour),
		Source:      fmt.Sprintf("Source %d", rand.Intn(5)),
		Severity:    []string{"Low", "Medium", "High"}[rand.Intn(3)],
	}
}

func insertEvent(e app.Event) error {
	_, err := db.Exec(`
		INSERT INTO events (name, description, timestamp, source, severity)
		VALUES (?, ?, ?, ?, ?)
	`, e.Name, e.Description, e.Timestamp, e.Source, e.Severity)
	return err
}
