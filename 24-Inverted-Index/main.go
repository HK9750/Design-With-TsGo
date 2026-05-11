package main

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// tokenPattern matches sequences of lowercase letters and digits, used to
// extract tokens from document text.
var tokenPattern = regexp.MustCompile(`[a-z0-9]+`)

// InvertedIndex maps terms to document IDs with term frequency counts.
// It enables fast full-text search across a corpus of documents.
// Time complexity: O(1) average per term lookup, O(d * t) to add a document
// where d is distinct terms and t is average term length.
type InvertedIndex struct{ postings map[string]map[string]int }

// NewInvertedIndex creates an empty inverted index with initialized postings map.
// Time complexity: O(1).
func NewInvertedIndex() *InvertedIndex {
	log.Debug("Created InvertedIndex")
	return &InvertedIndex{postings: make(map[string]map[string]int)}
}

// AddDocument indexes a document by tokenizing its text and recording term
// frequencies per document ID.
// Time complexity: O(n + d) where n is the token count and d is distinct terms.
func (idx *InvertedIndex) AddDocument(id, text string) {
	defer log.Operation("InvertedIndex.AddDocument", "id=%s text_len=%d", id, len(text))()
	counts := make(map[string]int)
	for _, term := range tokenize(text) {
		counts[term]++
	}
	for term, count := range counts {
		if idx.postings[term] == nil {
			idx.postings[term] = make(map[string]int)
		}
		idx.postings[term][id] = count
	}
	log.Debug("Indexed %d distinct terms for document %s", len(counts), id)
}

// SearchAll performs an AND-conjunctive search over all terms in the query.
// It starts with the posting list of the first term and filters by each
// subsequent term, keeping only documents that contain ALL terms. Returns
// document IDs in sorted order, or nil if no terms match.
// Time complexity: O(t * p) where t is the number of query terms and p is
// the average posting list size.
func (idx *InvertedIndex) SearchAll(query string) []string {
	defer log.Operation("InvertedIndex.SearchAll", "query=%q", query)()
	terms := tokenize(query)
	if len(terms) == 0 {
		log.Warn("Empty query — no tokens extracted")
		return nil
	}
	log.Debug("Query tokens: %v", terms)
	result := make(map[string]bool)
	if idx.postings[terms[0]] == nil {
		log.Warn("Term %q not found in any document", terms[0])
		return nil
	}
	for id := range idx.postings[terms[0]] {
		result[id] = true
	}
	for _, term := range terms[1:] {
		if idx.postings[term] == nil {
			log.Info("Term %q not in index — empty result", term)
			return nil
		}
		for id := range result {
			if _, ok := idx.postings[term][id]; !ok {
				delete(result, id)
			}
		}
	}
	ids := make([]string, 0, len(result))
	for id := range result {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	log.Info("Found %d matching documents", len(ids))
	return ids
}

// tokenize lowercases the input text and extracts all token sequences matching
// the tokenPattern regex. Returns a slice of tokens in order of appearance.
func tokenize(text string) []string { return tokenPattern.FindAllString(strings.ToLower(text), -1) }

func main() {
	defer log.Operation("main", "Running Inverted Index demo")()
	defer log.Info("Inverted Index demo completed")

	logger.Section("Inverted Index — Full-Text Search Engine")
	log.Info("Simulating a search engine indexing technical documentation")
	log.Info("Each document is tokenized and indexed with term frequencies")

	idx := NewInvertedIndex()

	logger.Section("Document Ingestion — Indexing Articles")
	docs := map[string]string{
		"doc1": "distributed systems design patterns for scalable services",
		"doc2": "systems programming with Rust and Go concurrency",
		"doc3": "design patterns for distributed programming in Rust",
		"doc4": "scalable microservices design with Go",
	}
	start := time.Now()
	for id, text := range docs {
		idx.AddDocument(id, text)
	}
	log.Info("Indexed %d documents in %v", len(docs), time.Since(start))

	logger.Section("Query Execution — Full-Text Search")
	queries := []string{
		"systems design",
		"distributed programming",
		"Rust Go",
		"scalable services",
		"machine learning",
	}
	for _, q := range queries {
		start = time.Now()
		results := idx.SearchAll(q)
		duration := time.Since(start)
		if len(results) == 0 {
			log.Info("Query %q: no results in %v", q, duration)
		} else {
			log.Info("Query %q: %d results — %v in %v", q, len(results), results, duration)
		}
	}

	logger.Section("Stats Summary")
	logger.KeyValue("total_documents", len(docs))
	logger.KeyValue("total_terms_indexed", len(idx.postings))
}
