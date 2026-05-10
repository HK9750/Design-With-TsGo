package main

import "fmt"

type PrimaryReplicaStore struct {
	primary  map[string]string
	replicas []map[string]string
}

func NewPrimaryReplicaStore(replicaCount int) *PrimaryReplicaStore {
	replicas := make([]map[string]string, replicaCount)
	for i := range replicas {
		replicas[i] = make(map[string]string)
	}
	return &PrimaryReplicaStore{primary: make(map[string]string), replicas: replicas}
}

func (s *PrimaryReplicaStore) Write(key, value string) {
	s.primary[key] = value
	for _, replica := range s.replicas {
		replica[key] = value
	}
}

func (s *PrimaryReplicaStore) Read(key string, fromReplica bool) (string, bool) {
	if fromReplica && len(s.replicas) > 0 {
		value, ok := s.replicas[0][key]
		return value, ok
	}
	value, ok := s.primary[key]
	return value, ok
}

func (s *PrimaryReplicaStore) Failover(replicaIndex int) bool {
	if replicaIndex < 0 || replicaIndex >= len(s.replicas) {
		return false
	}
	s.primary = make(map[string]string)
	for key, value := range s.replicas[replicaIndex] {
		s.primary[key] = value
	}
	return true
}

func main() {
	store := NewPrimaryReplicaStore(2)
	store.Write("x", "1")
	value, _ := store.Read("x", true)
	fmt.Println(value)
}
