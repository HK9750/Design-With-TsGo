package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type InvertedIndex struct{ postings map[string]map[string]int }

func NewInvertedIndex() *InvertedIndex {
	return &InvertedIndex{postings: make(map[string]map[string]int)}
}

func (idx *InvertedIndex) AddDocument(id, text string) {
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
}

func (idx *InvertedIndex) SearchAll(query string) []string {
	terms := tokenize(query)
	if len(terms) == 0 {
		return nil
	}
	result := make(map[string]bool)
	for id := range idx.postings[terms[0]] {
		result[id] = true
	}
	for _, term := range terms[1:] {
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
	return ids
}

var tokenPattern = regexp.MustCompile(`[a-z0-9]+`)

func tokenize(text string) []string { return tokenPattern.FindAllString(strings.ToLower(text), -1) }

func main() {
	idx := NewInvertedIndex()
	idx.AddDocument("1", "distributed systems design")
	idx.AddDocument("2", "systems programming")
	fmt.Println(idx.SearchAll("systems design"))
}
