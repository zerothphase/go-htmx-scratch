package main

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"
)

type Event struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Timestamp   time.Time `json:"timestamp"`
	Description string    `json:"description"`
	Source      string    `json:"source"`
	Severity    string    `json:"severity"`
}

const (
	eventsPerPage = 10
)

var db *sql.DB

func main() {
	// Initialize database connection
	var err error
	db, err = sql.Open("sqlite3", "./events.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			description TEXT,
			source TEXT,
			severity TEXT
		);
	`)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/events", handleEvents)
	http.HandleFunc("/filter", handleFilter)
	http.ListenAndServe(":8080", nil)
}

func handleEvents(w http.ResponseWriter, r *http.Request) {
	showDescription := r.URL.Query().Get("show-description") == "on"
	showSource := r.URL.Query().Get("show-source") == "on"
	showSeverity := r.URL.Query().Get("show-severity") == "on"
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	events, totalCount, err := getEvents(page, showDescription, showSource, showSeverity)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderEventsTable(w, events, showDescription, showSource, showSeverity, page, totalCount)
}

func handleFilter(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("filter")
	showDescription := r.URL.Query().Get("show-description") == "on"
	showSource := r.URL.Query().Get("show-source") == "on"
	showSeverity := r.URL.Query().Get("show-severity") == "on"
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	events, totalCount, err := getFilteredEvents(filter, page, showDescription, showSource, showSeverity)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderEventsTable(w, events, showDescription, showSource, showSeverity, page, totalCount)
}

func getEvents(page int, showDescription, showSource, showSeverity bool) ([]Event, int, error) {
	offset := (page - 1) * eventsPerPage

	// Get total count
	var totalCount int
	err := db.QueryRow("SELECT COUNT(*) FROM events").Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated events
	query := "SELECT id, name, timestamp"
	if showDescription {
		query += ", description"
	}
	if showSource {
		query += ", source"
	}
	if showSeverity {
		query += ", severity"
	}
	query += " FROM events ORDER BY timestamp DESC LIMIT ? OFFSET ?"

	rows, err := db.Query(query, eventsPerPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		var scanArgs []interface{}
		scanArgs = append(scanArgs, &e.ID, &e.Name, &e.Timestamp)
		if showDescription {
			scanArgs = append(scanArgs, &e.Description)
		}
		if showSource {
			scanArgs = append(scanArgs, &e.Source)
		}
		if showSeverity {
			scanArgs = append(scanArgs, &e.Severity)
		}
		err := rows.Scan(scanArgs...)
		if err != nil {
			return nil, 0, err
		}
		events = append(events, e)
	}

	return events, totalCount, nil
}

func getFilteredEvents(filter string, page int, showDescription, showSource, showSeverity bool) ([]Event, int, error) {
	offset := (page - 1) * eventsPerPage

	// Get total count
	var totalCount int
	err := db.QueryRow("SELECT COUNT(*) FROM events WHERE timestamp >= ?", filter).Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated filtered events
	query := "SELECT id, name, timestamp"
	if showDescription {
		query += ", description"
	}
	if showSource {
		query += ", source"
	}
	if showSeverity {
		query += ", severity"
	}
	query += " FROM events WHERE timestamp >= ? ORDER BY timestamp DESC LIMIT ? OFFSET ?"

	rows, err := db.Query(query, filter, eventsPerPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		var scanArgs []interface{}
		scanArgs = append(scanArgs, &e.ID, &e.Name, &e.Timestamp)
		if showDescription {
			scanArgs = append(scanArgs, &e.Description)
		}
		if showSource {
			scanArgs = append(scanArgs, &e.Source)
		}
		if showSeverity {
			scanArgs = append(scanArgs, &e.Severity)
		}
		err := rows.Scan(scanArgs...)
		if err != nil {
			return nil, 0, err
		}
		events = append(events, e)
	}

	return events, totalCount, nil
}

func renderEventsTable(w http.ResponseWriter, events []Event, showDescription, showSource, showSeverity bool, currentPage, totalCount int) {
	w.Header().Set("Content-Type", "text/html")

	// Render table
	w.Write([]byte(`
		<table class="w-full bg-white shadow-md rounded mb-4">
			<thead>
				<tr class="bg-gray-200 text-gray-600 uppercase text-sm leading-normal">
					<th class="py-3 px-6 text-left">ID</th>
					<th class="py-3 px-6 text-left">Name</th>
					<th class="py-3 px-6 text-left">Timestamp</th>
	`))

	if showDescription {
		w.Write([]byte(`<th class="py-3 px-6 text-left">Description</th>`))
	}
	if showSource {
		w.Write([]byte(`<th class="py-3 px-6 text-left">Source</th>`))
	}
	if showSeverity {
		w.Write([]byte(`<th class="py-3 px-6 text-left">Severity</th>`))
	}

	w.Write([]byte(`
				</tr>
			</thead>
			<tbody class="text-gray-600 text-sm font-light">
	`))

	for _, e := range events {
		w.Write([]byte(`
			<tr class="border-b border-gray-200 hover:bg-gray-100">
				<td class="py-3 px-6 text-left whitespace-nowrap">` + strconv.FormatInt(e.ID, 10) + `</td>
				<td class="py-3 px-6 text-left">` + e.Name + `</td>
				<td class="py-3 px-6 text-left">` + e.Timestamp.Format(time.RFC3339) + `</td>
		`))

		if showDescription {
			w.Write([]byte(`<td class="py-3 px-6 text-left">` + e.Description + `</td>`))
		}
		if showSource {
			w.Write([]byte(`<td class="py-3 px-6 text-left">` + e.Source + `</td>`))
		}
		if showSeverity {
			w.Write([]byte(`<td class="py-3 px-6 text-left">` + e.Severity + `</td>`))
		}

		w.Write([]byte(`</tr>`))
	}

	w.Write([]byte(`
			</tbody>
		</table>
	`))

	// Add pagination controls (unchanged)
	// ...
}
