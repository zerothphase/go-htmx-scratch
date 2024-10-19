package main

import (
	"database/sql"
	"html/template"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/zerothphase/go-htmx-scratch/internal/app"
)

const eventsPerPage = 25

var db *sql.DB
var tmpl *template.Template

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./events.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	initDB()

	tmpl, err = template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatal(err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/", handleIndex)
	r.HandleFunc("/events", handleEvents)
	r.HandleFunc("/filter", handleFilter)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func initDB() {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			description TEXT,
			timestamp DATETIME,
			source TEXT,
			severity TEXT
		)
	`)
	if err != nil {
		log.Fatal(err)
	}
}

// Implement handleIndex, handleEvents, and handleFilter functions here

func handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl.Execute(w, nil)
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

func getEvents(page int, showDescription, showSource, showSeverity bool) ([]app.Event, int, error) {
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

	var events []app.Event
	for rows.Next() {
		var e app.Event
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

func getFilteredEvents(filter string, page int, showDescription, showSource, showSeverity bool) ([]app.Event, int, error) {
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

	var events []app.Event
	for rows.Next() {
		var e app.Event
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

func renderEventsTable(w http.ResponseWriter, events []app.Event, showDescription, showSource, showSeverity bool, currentPage, totalCount int) {
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

	// Add pagination controls
	totalPages := int(math.Ceil(float64(totalCount) / float64(eventsPerPage)))
	w.Write([]byte(`
		<div class="flex justify-between items-center mt-4">
			<div>
				Showing page ` + strconv.Itoa(currentPage) + ` of ` + strconv.Itoa(totalPages) + `
			</div>
			<div>
	`))

	if currentPage > 1 {
		w.Write([]byte(`
			<button class="bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded mr-2"
				hx-get="/events?page=` + strconv.Itoa(currentPage-1) + `"
				hx-target="#events-table"
				hx-swap="innerHTML">
				Previous
			</button>
		`))
	}

	if currentPage < totalPages {
		w.Write([]byte(`
			<button class="bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded"
				hx-get="/events?page=` + strconv.Itoa(currentPage+1) + `"
				hx-target="#events-table"
				hx-swap="innerHTML">
				Next
			</button>
		`))
	}

	w.Write([]byte(`
			</div>
		</div>
	`))
}
