package main

import (
	"database/sql"
	"html/template"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zerothphase/go-htmx-scratch/internal/app"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB
var tmpl *template.Template

const eventsPerPage = 50

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./events.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Define custom template functions
	funcMap := template.FuncMap{
		"lower": strings.ToLower,
	}

	// Parse the template with the custom function map
	tmpl, err = template.New("index.html").Funcs(funcMap).ParseFiles("templates/index.html")
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

func handleIndex(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Columns []app.Column
	}{
		Columns: app.AvailableColumns,
	}
	tmpl.Execute(w, data)
}

func handleEvents(w http.ResponseWriter, r *http.Request) {
	columns := getSelectedColumns(r)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	events, totalCount, err := getEvents(page, columns)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderEventsTable(w, events, columns, page, totalCount)
}

func handleFilter(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("filter")
	columns := getSelectedColumns(r)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	events, totalCount, err := getFilteredEvents(filter, page, columns)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderEventsTable(w, events, columns, page, totalCount)
}

func getSelectedColumns(r *http.Request) []app.Column {
	selectedColumns := make(map[string]app.Column)

	// First, add all default columns
	for _, col := range app.GetDefaultColumns() {
		selectedColumns[col.Name] = col
	}

	// Then, add any additional selected columns
	for _, col := range app.AvailableColumns {
		if r.URL.Query().Get("show-"+strings.ToLower(col.Name)) == "on" {
			selectedColumns[col.Name] = col
		}
	}

	// Convert the map to a slice, preserving the original order
	var result []app.Column
	for _, col := range app.AvailableColumns {
		if _, ok := selectedColumns[col.Name]; ok {
			result = append(result, col)
		}
	}

	return result
}

func getEvents(page int, columns []app.Column) ([]app.Event, int, error) {
	offset := (page - 1) * eventsPerPage

	var totalCount int
	err := db.QueryRow("SELECT COUNT(*) FROM events").Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}

	query := buildSelectQuery(columns)
	query += " FROM events ORDER BY timestamp DESC LIMIT ? OFFSET ?"

	rows, err := db.Query(query, eventsPerPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	events, err := scanEvents(rows, columns)
	if err != nil {
		return nil, 0, err
	}

	return events, totalCount, nil
}

func getFilteredEvents(filter string, page int, columns []app.Column) ([]app.Event, int, error) {
	offset := (page - 1) * eventsPerPage

	var totalCount int
	err := db.QueryRow("SELECT COUNT(*) FROM events WHERE timestamp >= ?", filter).Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}

	query := buildSelectQuery(columns)
	query += " FROM events WHERE timestamp >= ? ORDER BY timestamp DESC LIMIT ? OFFSET ?"

	rows, err := db.Query(query, filter, eventsPerPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	events, err := scanEvents(rows, columns)
	if err != nil {
		return nil, 0, err
	}

	return events, totalCount, nil
}

func buildSelectQuery(columns []app.Column) string {
	var fields []string
	for _, col := range columns {
		fields = append(fields, col.DBField)
	}
	return "SELECT " + strings.Join(fields, ", ")
}

func scanEvents(rows *sql.Rows, columns []app.Column) ([]app.Event, error) {
	var events []app.Event
	for rows.Next() {
		var e app.Event
		var scanArgs []interface{}
		for _, col := range columns {
			switch col.Name {
			case "ID":
				scanArgs = append(scanArgs, &e.ID)
			case "Name":
				scanArgs = append(scanArgs, &e.Name)
			case "Description":
				scanArgs = append(scanArgs, &e.Description)
			case "Timestamp":
				scanArgs = append(scanArgs, &e.Timestamp)
			case "Source":
				scanArgs = append(scanArgs, &e.Source)
			case "Severity":
				scanArgs = append(scanArgs, &e.Severity)
			}
		}
		err := rows.Scan(scanArgs...)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

func renderEventsTable(w http.ResponseWriter, events []app.Event, columns []app.Column, currentPage, totalCount int) {
	w.Header().Set("Content-Type", "text/html")

	// Render table header
	w.Write([]byte(`
		<table class="w-full bg-white shadow-md rounded mb-4">
			<thead>
				<tr class="bg-gray-200 text-gray-600 uppercase text-sm leading-normal">
	`))

	for _, col := range columns {
		w.Write([]byte(`<th class="py-3 px-6 text-left">` + col.Name + `</th>`))
	}

	w.Write([]byte(`
				</tr>
			</thead>
			<tbody class="text-gray-600 text-sm font-light">
	`))

	// Render table rows
	for _, e := range events {
		w.Write([]byte(`<tr class="border-b border-gray-200 hover:bg-gray-100">`))
		for _, col := range columns {
			w.Write([]byte(`<td class="py-3 px-6 text-left">`))
			switch col.Name {
			case "ID":
				w.Write([]byte(strconv.FormatInt(e.ID, 10)))
			case "Name":
				w.Write([]byte(e.Name))
			case "Description":
				w.Write([]byte(e.Description))
			case "Timestamp":
				w.Write([]byte(e.Timestamp.Format(time.RFC3339)))
			case "Source":
				w.Write([]byte(e.Source))
			case "Severity":
				w.Write([]byte(e.Severity))
			}
			w.Write([]byte(`</td>`))
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
