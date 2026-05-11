package main

import (
	"fmt"
	"os"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// Document is a flexible schema-less document represented as a string-keyed
// map of arbitrary values.
type Document map[string]any

// DocumentStore provides schema-less document storage with secondary indexing.
// Indexes can be created on any field, enabling fast lookups without scanning
// the entire collection. When no index exists, queries fall back to a full
// collection scan.
// Time complexity: O(1) upsert, O(1) indexed find, O(n) full-scan find.
type DocumentStore struct {
	docs    map[string]Document
	indexes map[string]map[string]map[string]bool
}

// NewDocumentStore creates an empty document store with no indexes.
// Time complexity: O(1).
func NewDocumentStore() *DocumentStore {
	log.Debug("Created DocumentStore")
	return &DocumentStore{
		docs:    make(map[string]Document),
		indexes: make(map[string]map[string]map[string]bool),
	}
}

// CreateIndex builds a secondary index on the specified field across all
// existing documents. The index maps field values to sets of document IDs.
// Time complexity: O(n) where n is the number of existing documents.
func (s *DocumentStore) CreateIndex(field string) {
	defer log.Operation("DocumentStore.CreateIndex", "field=%s", field)()
	index := make(map[string]map[string]bool)
	for id, doc := range s.docs {
		addToIndex(index, fmt.Sprint(doc[field]), id)
	}
	s.indexes[field] = index
	log.Info("Created index on field %q with %d unique values", field, len(index))
}

// Upsert inserts or updates a document by ID. All existing indexes are updated
// to reflect the new field values.
// Time complexity: O(i) where i is the number of indexes.
func (s *DocumentStore) Upsert(id string, doc Document) {
	defer log.Operation("DocumentStore.Upsert", "id=%s fields=%d", id, len(doc))()
	s.docs[id] = doc
	for field, index := range s.indexes {
		addToIndex(index, fmt.Sprint(doc[field]), id)
	}
	log.Debug("Upserted document %s", id)
}

// FindByField searches for documents where the given field matches the value.
// If an index exists on the field, it performs an O(1) index lookup; otherwise
// it falls back to a full collection scan O(n).
// Time complexity: O(1) with index, O(n) without index.
func (s *DocumentStore) FindByField(field string, value any) []Document {
	defer log.Operation("DocumentStore.FindByField", "field=%s value=%v", field, value)()
	var result []Document
	if index := s.indexes[field]; index != nil {
		log.Debug("Using index on %q", field)
		for id := range index[fmt.Sprint(value)] {
			result = append(result, s.docs[id])
		}
		log.Info("Indexed lookup found %d documents", len(result))
		return result
	}
	log.Debug("No index on %q, performing full scan", field)
	for _, doc := range s.docs {
		if fmt.Sprint(doc[field]) == fmt.Sprint(value) {
			result = append(result, doc)
		}
	}
	log.Info("Full scan found %d documents", len(result))
	return result
}

// addToIndex inserts a document ID into the given index under the specified
// value. Creates the value bucket if it does not exist.
// Time complexity: O(1).
func addToIndex(index map[string]map[string]bool, value string, id string) {
	if index[value] == nil {
		index[value] = make(map[string]bool)
	}
	index[value][id] = true
}

func main() {
	defer log.Operation("main", "Running Document Store demo")()
	defer log.Info("Document Store demo completed")

	logger.Section("Document Store — User Profile Database")
	log.Info("Simulating a user management system with indexed lookups")
	log.Info("Documents are schema-less; indexes accelerate queries by field")

	store := NewDocumentStore()

	logger.Section("Ingesting User Profiles")
	start := time.Now()
	store.Upsert("u1", Document{"type": "user", "role": "admin", "name": "Ada Lovelace", "dept": "engineering"})
	store.Upsert("u2", Document{"type": "user", "role": "developer", "name": "Grace Hopper", "dept": "engineering"})
	store.Upsert("u3", Document{"type": "user", "role": "manager", "name": "Alan Turing", "dept": "research"})
	store.Upsert("u4", Document{"type": "service_account", "role": "bot", "name": "ci-builder", "dept": "infra"})
	log.Info("Inserted %d documents in %v", 4, time.Since(start))

	logger.Section("Secondary Index Creation")
	log.Info("Creating indexes for accelerated queries...")
	store.CreateIndex("type")
	store.CreateIndex("dept")
	store.CreateIndex("role")
	log.Info("Indexes created on: type, dept, role")

	logger.Section("Query: Find All Users by Type (Indexed)")
	users := store.FindByField("type", "user")
	log.Info("Found %d documents of type 'user':", len(users))
	for _, doc := range users {
		log.Info("  %s (%s)", doc["name"], doc["role"])
	}

	logger.Section("Query: Find by Department (Indexed)")
	engineers := store.FindByField("dept", "engineering")
	log.Info("Found %d documents in engineering department:", len(engineers))
	for _, doc := range engineers {
		log.Info("  %s — %s", doc["name"], doc["role"])
	}

	logger.Section("Query: Find by Role (Indexed)")
	admins := store.FindByField("role", "admin")
	log.Info("Found %d admin(s):", len(admins))
	for _, doc := range admins {
		log.Info("  %s", doc["name"])
	}

	logger.Section("Edge Case — Query Without Index")
	store2 := NewDocumentStore()
	store2.Upsert("x1", Document{"color": "red"})
	store2.Upsert("x2", Document{"color": "blue"})
	result := store2.FindByField("color", "red")
	log.Info("Full-scan on 'color': found %d document(s)", len(result))

	logger.Section("Stats Summary")
	logger.KeyValue("total_documents", len(store.docs))
	logger.KeyValue("total_indexes", len(store.indexes))
	logger.KeyValue("indexed_fields", []string{"type", "dept", "role"})
}
