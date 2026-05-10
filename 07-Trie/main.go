package main

import "fmt"

type TrieNode struct {
	children map[rune]*TrieNode
	end      bool
	value    string
}

type Trie struct{ root *TrieNode }

func NewTrie() *Trie { return &Trie{root: &TrieNode{children: make(map[rune]*TrieNode)}} }

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
}

func (t *Trie) Search(word string) (string, bool) {
	node := t.findNode(word)
	if node == nil || !node.end {
		return "", false
	}
	return node.value, true
}

func (t *Trie) StartsWith(prefix string) bool { return t.findNode(prefix) != nil }

func (t *Trie) Delete(word string) bool { return t.deleteFrom(t.root, []rune(word), 0) }

func (t *Trie) WordsWithPrefix(prefix string) []string {
	node := t.findNode(prefix)
	if node == nil {
		return nil
	}
	var result []string
	t.collect(node, []rune(prefix), &result)
	return result
}

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

func (t *Trie) collect(node *TrieNode, word []rune, result *[]string) {
	if node.end {
		*result = append(*result, string(word))
	}
	for char, child := range node.children {
		t.collect(child, append(word, char), result)
	}
}

func main() {
	trie := NewTrie()
	trie.Insert("system", "system")
	trie.Insert("syscall", "syscall")
	value, ok := trie.Search("system")
	fmt.Println(value, ok, trie.WordsWithPrefix("sys"))
}
