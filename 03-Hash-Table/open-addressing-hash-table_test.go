package main

import (
	"fmt"
	"testing"
)

func TestOpenAddressingHashTableBasicOperations(t *testing.T) {
	ht := NewOpenAddressingHashTable[int](0)

	ht.Set("one", 1)
	ht.Set("two", 2)
	ht.Set("three", 3)

	if value, ok := ht.Get("one"); !ok || value != 1 {
		t.Fatalf("expected one=1, got value=%d ok=%t", value, ok)
	}
	if _, ok := ht.Get("missing"); ok {
		t.Fatal("expected missing key to be absent")
	}

	ht.Set("one", 100)
	if value, ok := ht.Get("one"); !ok || value != 100 {
		t.Fatalf("expected updated one=100, got value=%d ok=%t", value, ok)
	}

	if !ht.Delete("two") {
		t.Fatal("expected delete of existing key to return true")
	}
	if _, ok := ht.Get("two"); ok {
		t.Fatal("expected deleted key to be absent")
	}
	if ht.Delete("two") {
		t.Fatal("expected second delete to return false")
	}

	ht.Set("two", 222)
	if value, ok := ht.Get("two"); !ok || value != 222 {
		t.Fatalf("expected reinserted two=222, got value=%d ok=%t", value, ok)
	}
}

func TestOpenAddressingHashTableResizeAndTombstones(t *testing.T) {
	ht := NewOpenAddressingHashTable[int](16)
	largeCount := 10000

	for i := 0; i < largeCount; i++ {
		ht.Set(fmt.Sprintf("key_%d", i), i)
	}
	for i := 0; i < largeCount; i++ {
		key := fmt.Sprintf("key_%d", i)
		value, ok := ht.Get(key)
		if !ok || value != i {
			t.Fatalf("expected %s=%d, got value=%d ok=%t", key, i, value, ok)
		}
	}

	for i := 0; i < largeCount-100; i++ {
		if !ht.Delete(fmt.Sprintf("key_%d", i)) {
			t.Fatalf("expected key_%d to delete", i)
		}
	}
	for i := largeCount - 100; i < largeCount; i++ {
		key := fmt.Sprintf("key_%d", i)
		value, ok := ht.Get(key)
		if !ok || value != i {
			t.Fatalf("expected retained %s=%d, got value=%d ok=%t", key, i, value, ok)
		}
	}

	for i := 0; i < 1000; i++ {
		ht.Set(fmt.Sprintf("reused_%d", i), i)
	}
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("reused_%d", i)
		value, ok := ht.Get(key)
		if !ok || value != i {
			t.Fatalf("expected reused %s=%d, got value=%d ok=%t", key, i, value, ok)
		}
	}

	if ht.Size() != 1100 {
		t.Fatalf("expected size 1100, got %d", ht.Size())
	}
	if ht.Capacity() < 16 {
		t.Fatalf("capacity dropped below minimum: %d", ht.Capacity())
	}
}
