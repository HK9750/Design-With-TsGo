package main

import "fmt"

type ColumnStore struct {
	columns map[string][]string
	rows    int
}

func NewColumnStore() *ColumnStore { return &ColumnStore{columns: make(map[string][]string)} }

func (s *ColumnStore) Insert(row map[string]string) {
	for column := range s.columns {
		s.columns[column] = append(s.columns[column], row[column])
	}
	for column, value := range row {
		if _, ok := s.columns[column]; !ok {
			s.columns[column] = make([]string, s.rows)
			s.columns[column] = append(s.columns[column], value)
		}
	}
	s.rows++
}

func (s *ColumnStore) Project(columns []string) []map[string]string {
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
	return result
}

func main() {
	store := NewColumnStore()
	store.Insert(map[string]string{"id": "1", "country": "PK"})
	store.Insert(map[string]string{"id": "2", "country": "US"})
	fmt.Println(store.Project([]string{"country"}))
}
