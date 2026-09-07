package main

import "net/http"

func (s *Store) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.flags[key]; !ok {
		return false
	}
	delete(s.flags, key)
	return true
}

func handleDeleteFlag(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if !store.Delete(key) {
		writeError(w, http.StatusNotFound, "flag not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
