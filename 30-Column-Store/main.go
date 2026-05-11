package main

import (
	"os"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// ColumnStore is a column-oriented data store where data is organized by
// column rather than by row. This layout is optimized for analytical workloads
// that scan a subset of columns across many rows (OLAP). Each column stores
// a contiguous slice of string values.
// Space complexity: O(r * c) where r is the number of rows and c is columns.
type ColumnStore struct {
	columns map[string][]string
	rows    int
}

// NewColumnStore creates an empty column store with no columns or rows.
// Time complexity: O(1).
func NewColumnStore() *ColumnStore {
	log.Debug("Created ColumnStore")
	return &ColumnStore{columns: make(map[string][]string)}
}

// Insert adds a row of string-encoded values to the store. For each existing
// column, a null placeholder (empty string) is added; for new columns, the
// column is backfilled with empty strings for prior rows before appending
// the value.
// Time complexity: O(c) where c is the number of columns.
func (s *ColumnStore) Insert(row map[string]string) {
	defer log.Operation("ColumnStore.Insert", "row=%v", row)()
	// Pad existing columns that are not in the new row
	for column := range s.columns {
		if _, present := row[column]; !present {
			log.Debug("Padding column %q with empty string", column)
		}
		s.columns[column] = append(s.columns[column], row[column])
	}
	// Add new columns introduced by this row
	for column, value := range row {
		if _, ok := s.columns[column]; !ok {
			log.Debug("Adding new column %q, backfilling %d rows", column, s.rows)
			s.columns[column] = make([]string, s.rows)
			s.columns[column] = append(s.columns[column], value)
		}
	}
	s.rows++
}

// Project extracts a subset of columns across all rows and returns them as a
// slice of row maps. This is the primary read path for analytical queries.
// Time complexity: O(r * p) where r is the number of rows and p is the number
// of projected columns.
func (s *ColumnStore) Project(columns []string) []map[string]string {
	defer log.Operation("ColumnStore.Project", "columns=%v rows=%d", columns, s.rows)()
	result := make([]map[string]string, s.rows)
	for i := 0; i < s.rows; i++ {
		row := make(map[string]string)
		for _, column := range columns {
			if values := s.columns[column]; i < len(values) {
				row[column] = values[i]
			}
		}
		result[i] = row
	}
	log.Info("Projection returned %d rows across %d columns", s.rows, len(columns))
	return result
}

func main() {
	defer log.Operation("main", "Running Column Store demo")()
	defer log.Info("Column Store demo completed")

	logger.Section("Column Store — Real-Time Analytics Engine")
	log.Info("Simulating an analytics database for a SaaS billing platform")
	log.Info("Columnar layout optimizes aggregate queries (SUM, COUNT, AVG) over large row sets")

	store := NewColumnStore()

	logger.Section("Data Ingestion — Streaming Invoice Events")
	start := time.Now()
	store.Insert(map[string]string{
		"invoice_id":  "INV-001",
		"customer_id": "C42",
		"country":     "PK",
		"amount":      "150.00",
		"currency":    "USD",
		"status":      "paid",
	})
	store.Insert(map[string]string{
		"invoice_id":  "INV-002",
		"customer_id": "C17",
		"country":     "US",
		"amount":      "320.00",
		"currency":    "USD",
		"status":      "pending",
	})
	store.Insert(map[string]string{
		"invoice_id":  "INV-003",
		"customer_id": "C42",
		"country":     "PK",
		"amount":      "89.50",
		"currency":    "USD",
		"status":      "paid",
	})
	store.Insert(map[string]string{
		"invoice_id":  "INV-004",
		"customer_id": "C99",
		"country":     "UK",
		"amount":      "210.00",
		"currency":    "GBP",
		"status":      "paid",
	})
	store.Insert(map[string]string{
		"invoice_id":  "INV-005",
		"customer_id": "C17",
		"country":     "US",
		"amount":      "75.00",
		"currency":    "USD",
		"status":      "overdue",
	})
	log.Info("Ingested %d invoices in %v", 5, time.Since(start))

	logger.Section("Analytical Query 1 — Revenue by Country (Project + Aggregate)")
	log.Info("Projecting 'country' and 'amount' columns for revenue analysis")
	result := store.Project([]string{"country", "amount"})
	for i, row := range result {
		log.Info("  Invoice[%d]: country=%s amount=%s", i, row["country"], row["amount"])
	}

	logger.Section("Analytical Query 2 — All Invoice IDs (Single Column Projection)")
	invoices := store.Project([]string{"invoice_id"})
	log.Info("Invoice IDs across all %d rows:", len(invoices))
	for _, row := range invoices {
		log.Info("  %s", row["invoice_id"])
	}

	logger.Section("Analytical Query 3 — Full Row Reconstruction")
	log.Info("Projecting all columns (simulates SELECT *)")
	fullResult := store.Project([]string{"invoice_id", "customer_id", "country", "amount", "currency", "status"})
	log.Info("Reconstructed %d full rows", len(fullResult))
	for _, row := range fullResult {
		log.Info("  %s | %s | %s | %s | %s | %s",
			row["invoice_id"], row["customer_id"], row["country"],
			row["amount"], row["currency"], row["status"])
	}

	logger.Section("Edge Case — Partial Data Insert (Sparse Columns)")
	store2 := NewColumnStore()
	store2.Insert(map[string]string{"event": "login", "user": "alice"})
	store2.Insert(map[string]string{"event": "purchase", "user": "bob", "item": "book"})
	store2.Insert(map[string]string{"event": "logout", "user": "alice", "duration_ms": "1200"})
	log.Info("Sparse events table — columns: %v", store2.Project([]string{"event", "user", "item", "duration_ms"}))

	logger.Section("Stats Summary")
	logger.KeyValue("total_rows", store.rows)
	logger.KeyValue("total_columns", len(store.columns))
	logger.KeyValue("column_names", []string{"invoice_id", "customer_id", "country", "amount", "currency", "status"})
}
