package main

import (
	"encoding/json"
	"net/http"
	"sort"
)

func (s *Store) List() []Flag {
	s.mu.RLock()
	defer s.mu.RUnlock()

	flags := make([]Flag, 0, len(s.flags))
	for _, f := range s.flags {
		flags = append(flags, f)
	}
	sort.Slice(flags, func(i, j int) bool { return flags[i].Key < flags[j].Key })
	return flags
}

func (s *Store) Get(key string) (Flag, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	f, ok := s.flags[key]
	return f, ok
}

func handleListFlags(w http.ResponseWriter, r *http.Request) {
	flags := store.List()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(flags)
}

func handleGetFlag(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	f, ok := store.Get(key)
	if !ok {
		writeError(w, http.StatusNotFound, "flag not found")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(f)
}
