// Package main demonstrates a Trie (prefix tree) data structure for efficient
// string storage and prefix-based retrieval. Supports insert, search, delete,
// prefix matching, and word collection. Ideal for an autocomplete engine.
package main

import (
	"fmt"
	"os"
	"time"

	"design-with-tsgo/pkg/logger"
)

var log = logger.New(os.Stdout, logger.DEBUG, true)

// TrieNode represents a node in the trie. Each node holds a map of child nodes keyed by rune,
// an end-of-word flag, and the associated value for complete words.
type TrieNode struct {
	children map[rune]*TrieNode
	end      bool
	value    string
}

// Trie implements a prefix tree (trie) data structure. It provides O(m) operations
// where m is the length of the key string, making it ideal for autocomplete,
// spell checking, and IP routing prefix matching.
type Trie struct {
	root *TrieNode
}

// NewTrie creates an empty trie with an initialized root node.
// Time complexity: O(1).
func NewTrie() *Trie {
	log.Debug("Created trie")
	return &Trie{root: &TrieNode{children: make(map[rune]*TrieNode)}}
}

// Insert adds a word and its associated value into the trie. If the word already exists,
// its value is updated. Time complexity: O(m) where m is the word length.
func (t *Trie) Insert(word string, value string) {
	node := t.root
	for _, char := range word {
		if node.children[char] == nil {
			node.children[char] = &TrieNode{children: make(map[rune]*TrieNode)}
		}
		node = node.children[char]
	}
	node.end = true
	node.value = value
	log.Debug("Trie insert (word=%q, value=%q)", word, value)
}

// Search looks up a complete word in the trie. Returns the associated value and true if found,
// or empty string and false otherwise. Time complexity: O(m).
func (t *Trie) Search(word string) (string, bool) {
	node := t.findNode(word)
	if node == nil || !node.end {
		log.Debug("Trie search miss (word=%q)", word)
		return "", false
	}
	log.Debug("Trie search hit (word=%q, value=%q)", word, node.value)
	return node.value, true
}

// StartsWith returns true if any word in the trie starts with the given prefix.
// Time complexity: O(m) where m is the prefix length.
func (t *Trie) StartsWith(prefix string) bool {
	found := t.findNode(prefix) != nil
	log.Debug("Trie starts_with (prefix=%q, result=%v)", prefix, found)
	return found
}

// Delete removes a word from the trie. Returns true if the word was found and removed.
// Cleans up child nodes that are no longer needed. Time complexity: O(m).
func (t *Trie) Delete(word string) bool {
	result := t.deleteFrom(t.root, []rune(word), 0)
	if result {
		log.Debug("Trie delete (word=%q)", word)
	} else {
		log.Warn("Trie delete miss (word=%q)", word)
	}
	return result
}

// WordsWithPrefix returns all words in the trie that start with the given prefix.
// Returns nil if no such words exist. Time complexity: O(m + k) where k is the number of matching words.
func (t *Trie) WordsWithPrefix(prefix string) []string {
	node := t.findNode(prefix)
	if node == nil {
		log.Debug("Trie words_with_prefix: none (prefix=%q)", prefix)
		return nil
	}
	var result []string
	t.collect(node, []rune(prefix), &result)
	log.Debug("Trie words_with_prefix (prefix=%q, count=%d)", prefix, len(result))
	return result
}

// findNode traverses the trie following the characters in text.
// Returns the node at the end of the path, or nil if the path does not exist.
func (t *Trie) findNode(text string) *TrieNode {
	node := t.root
	for _, char := range text {
		if node.children[char] == nil {
			return nil
		}
		node = node.children[char]
	}
	return node
}

// deleteFrom recursively removes a word from the trie starting at the given node.
// Cleans up empty non-terminal child nodes on the way back up.
func (t *Trie) deleteFrom(node *TrieNode, word []rune, index int) bool {
	if index == len(word) {
		if !node.end {
			return false
		}
		node.end = false
		node.value = ""
		return true
	}
	child := node.children[word[index]]
	if child == nil || !t.deleteFrom(child, word, index+1) {
		return false
	}
	if !child.end && len(child.children) == 0 {
		delete(node.children, word[index])
	}
	return true
}

// collect performs a depth-first traversal to gather all complete words under the given node.
func (t *Trie) collect(node *TrieNode, word []rune, result *[]string) {
	if node.end {
		*result = append(*result, string(word))
	}
	for char, child := range node.children {
		t.collect(child, append(word, char), result)
	}
}

// main demonstrates the trie as an autocomplete engine for a search bar,
// indexing commands and file paths with fast prefix-based suggestions.
func main() {
	defer log.Operation("main", "Running Trie demo")()

	logger.Section("Trie — Search Bar Autocomplete Engine")

	trie := NewTrie()

	logger.Section("Indexing system commands and file paths")
	commands := map[string]string{
		"system":    "system — display system information",
		"syscall":   "syscall — invoke a system call",
		"syslog":    "syslog — view system log entries",
		"sync":      "sync — synchronize cached writes to disk",
		"syntax":    "syntax — check file syntax",
		"sysctl":    "sysctl — configure kernel parameters",
		"systemd":   "systemd — system and service manager",
		"sysstat":   "sysstat — system performance monitoring",
		"container": "container — manage container runtimes",
		"config":    "config — manage configuration files",
		"compile":   "compile — build from source",
	}
	for word, desc := range commands {
		trie.Insert(word, desc)
	}
	log.Info("Indexed %d commands", len(commands))

	logger.Section("Exact match lookups")
	if value, ok := trie.Search("system"); ok {
		log.Info("system -> %s", value)
	}
	if value, ok := trie.Search("syscall"); ok {
		log.Info("syscall -> %s", value)
	}
	if _, ok := trie.Search("nonexistent"); !ok {
		log.Info("nonexistent -> command not found")
	}

	logger.Section("Prefix-based autocomplete (user types 'sys')")
	matches := trie.WordsWithPrefix("sys")
	log.Info("Autocomplete suggestions for 'sys': %v", matches)

	logger.Section("Prefix check (does any command start with 'con'?)")
	log.Info("Prefix 'con' exists: %v", trie.StartsWith("con"))
	conMatches := trie.WordsWithPrefix("con")
	log.Info("Autocomplete for 'con': %v", conMatches)

	logger.Section("Delete deprecated command")
	trie.Delete("syslog")
	if _, ok := trie.Search("syslog"); !ok {
		log.Info("Command 'syslog' successfully removed")
	}
	log.Info("Updated autocomplete for 'sys': %v", trie.WordsWithPrefix("sys"))

	logger.Section("Bulk indexing performance — 5,000 file paths")
	start := time.Now()
	for i := 0; i < 5000; i++ {
		path := fmt.Sprintf("/usr/local/bin/tool_%d", i)
		trie.Insert(path, fmt.Sprintf("path_%d_desc", i))
	}
	log.Info("Indexed 5,000 file paths in %v", time.Since(start))

	start = time.Now()
	prefixMatches := trie.WordsWithPrefix("/usr/local")
	log.Info("Prefix lookup '/usr/local' returned %d matches in %v", len(prefixMatches), time.Since(start))

	logger.Section("Final Stats")
	log.Info("All operations completed — O(m) insert/search/delete where m is key length")
}
