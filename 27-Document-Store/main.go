package main

import "fmt"

type Document map[string]any

type DocumentStore struct {
	docs    map[string]Document
	indexes map[string]map[string]map[string]bool
}

func NewDocumentStore() *DocumentStore {
	return &DocumentStore{docs: make(map[string]Document), indexes: make(map[string]map[string]map[string]bool)}
}

func (s *DocumentStore) CreateIndex(field string) {
	index := make(map[string]map[string]bool)
	for id, doc := range s.docs {
		addToIndex(index, fmt.Sprint(doc[field]), id)
	}
	s.indexes[field] = index
}

func (s *DocumentStore) Upsert(id string, doc Document) {
	s.docs[id] = doc
	for field, index := range s.indexes {
		addToIndex(index, fmt.Sprint(doc[field]), id)
	}
}

func (s *DocumentStore) FindByField(field string, value any) []Document {
	var result []Document
	if index := s.indexes[field]; index != nil {
		for id := range index[fmt.Sprint(value)] {
			result = append(result, s.docs[id])
		}
		return result
	}
	for _, doc := range s.docs {
		if fmt.Sprint(doc[field]) == fmt.Sprint(value) {
			result = append(result, doc)
		}
	}
	return result
}

func addToIndex(index map[string]map[string]bool, value string, id string) {
	if index[value] == nil {
		index[value] = make(map[string]bool)
	}
	index[value][id] = true
}

func main() {
	store := NewDocumentStore()
	store.Upsert("1", Document{"type": "user", "name": "Ada"})
	store.CreateIndex("type")
	fmt.Println(store.FindByField("type", "user"))
}
