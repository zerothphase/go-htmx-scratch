package main

import (
	"database/sql"
	"fmt"
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

	funcMap := template.FuncMap{
		"lower": strings.ToLower,
	}

	tmpl, err = template.New("index.html").Funcs(funcMap).ParseFiles("templates/index.html")
	if err != nil {
		log.Fatal(err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/", handleIndex)
	r.HandleFunc("/events", handleEvents)

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
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("handleEvents: %+v", r.Form)

	columns := getSelectedColumns(r)
	page, _ := strconv.Atoi(r.Form.Get("page"))
	if page < 1 {
		page = 1
	}

	filters := getFilters(r)
	events, totalCount, err := getFilteredEvents(page, columns, filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderEventsTable(w, events, columns, page, totalCount)
}

func getSelectedColumns(r *http.Request) []app.Column {
	selectedColumns := make(map[string]app.Column)

	for _, col := range app.GetDefaultColumns() {
		selectedColumns[col.Name] = col
	}

	for _, col := range app.AvailableColumns {
		if r.Form.Get("show-"+strings.ToLower(col.Name)) == "on" {
			selectedColumns[col.Name] = col
		}
	}

	var result []app.Column
	for _, col := range app.AvailableColumns {
		if _, ok := selectedColumns[col.Name]; ok {
			result = append(result, col)
		}
	}

	return result
}

type Filters struct {
	TimestampFilter string
	TimestampValue  time.Time
	TimestampEnd    time.Time
	SourceFilter    []string
	SeverityFilter  string
	NameFilter      []string
}

func getFilters(r *http.Request) Filters {
	filters := Filters{}

	filters.TimestampFilter = r.Form.Get("timestamp-filter")
	if filters.TimestampFilter != "" {
		timestampValue := r.Form.Get("timestamp-value")
		filters.TimestampValue, _ = time.Parse(time.RFC3339, timestampValue)

		if filters.TimestampFilter == "between" {
			timestampEnd := r.Form.Get("timestamp-value-end")
			filters.TimestampEnd, _ = time.Parse(time.RFC3339, timestampEnd)
		}
	}

	sourceFilter := r.Form.Get("source-filter")
	if sourceFilter != "" {
		filters.SourceFilter = strings.Split(sourceFilter, ",")
	}

	filters.SeverityFilter = r.Form.Get("severity-filter")

	nameFilter := r.Form.Get("name-filter")
	if nameFilter != "" {
		filters.NameFilter = strings.Split(nameFilter, ",")
	}

	return filters
}

func getFilteredEvents(page int, columns []app.Column, filters Filters) ([]app.Event, int, error) {
	offset := (page - 1) * eventsPerPage

	query := buildSelectQuery(columns)
	query += " FROM events WHERE 1=1"
	var args []interface{}

	if filters.TimestampFilter != "" {
		switch filters.TimestampFilter {
		case "before":
			query += " AND timestamp <= ?"
			args = append(args, filters.TimestampValue)
		case "after":
			query += " AND timestamp >= ?"
			args = append(args, filters.TimestampValue)
		case "between":
			query += " AND timestamp BETWEEN ? AND ?"
			args = append(args, filters.TimestampValue, filters.TimestampEnd)
		}
	}

	if len(filters.SourceFilter) > 0 {
		placeholders := make([]string, len(filters.SourceFilter))
		for i := range filters.SourceFilter {
			placeholders[i] = "?"
			args = append(args, filters.SourceFilter[i])
		}
		query += " AND source IN (" + strings.Join(placeholders, ",") + ")"
	}

	if filters.SeverityFilter != "" {
		query += " AND severity = ?"
		args = append(args, filters.SeverityFilter)
	}

	if len(filters.NameFilter) > 0 {
		placeholders := make([]string, len(filters.NameFilter))
		for i := range filters.NameFilter {
			placeholders[i] = "?"
			args = append(args, filters.NameFilter[i])
		}
		query += " AND name IN (" + strings.Join(placeholders, ",") + ")"
	}

	countQuery := "SELECT COUNT(*) FROM (" + query + ")"
	var totalCount int
	err := db.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}

	query += " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	args = append(args, eventsPerPage, offset)

	rows, err := db.Query(query, args...)
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

	totalPages := int(math.Ceil(float64(totalCount) / float64(eventsPerPage)))

	// Render pagination controls at the top
	w.Write([]byte(`
		<div class="flex justify-between items-center mb-4">
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
					hx-swap="innerHTML"
					hx-include="form">
					Previous
				</button>
			`))
	}

	if currentPage < totalPages {
		w.Write([]byte(`
				<button class="bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded"
					hx-get="/events?page=` + strconv.Itoa(currentPage+1) + `"
					hx-target="#events-table"
					hx-swap="innerHTML"
					hx-include="form">
					Next
				</button>
			`))
	}

	w.Write([]byte(`
			</div>
		</div>
	`))

	// Render table
	gridCols := len(columns)
	gridClass := fmt.Sprintf("grid-cols-%d", gridCols)

	w.Write([]byte(`
		<div class="w-full bg-white shadow-md rounded mb-4 overflow-x-auto">
			<div class="grid ` + gridClass + ` gap-x-4 bg-gray-200 text-gray-600 uppercase text-sm leading-normal">
	`))

	for _, col := range columns {
		w.Write([]byte(`<div class="py-3 px-6 text-left whitespace-nowrap overflow-hidden text-ellipsis">` + col.Name + `</div>`))
	}

	w.Write([]byte(`
			</div>
			<div class="text-gray-600 text-sm font-light">
	`))

	for _, e := range events {
		w.Write([]byte(`<div class="grid ` + gridClass + ` gap-x-4 border-b border-gray-200 hover:bg-gray-100">`))
		for _, col := range columns {
			w.Write([]byte(`<div class="py-3 px-6 text-left whitespace-nowrap overflow-hidden text-ellipsis">`))
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
			w.Write([]byte(`</div>`))
		}
		w.Write([]byte(`</div>`))
	}

	w.Write([]byte(`
			</div>
		</div>
	`))
}
