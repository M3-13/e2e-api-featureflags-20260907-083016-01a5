package main

import (
	"encoding/json"
	"hash/fnv"
	"net/http"
)

func (s *Store) Evaluate(key, user string) (bool, bool) {
	s.mu.RLock()
	flag, ok := s.flags[key]
	s.mu.RUnlock()
	if !ok {
		return false, false
	}

	h := fnv.New64a()
	_, _ = h.Write([]byte(key + ":" + user))
	bucket := int(h.Sum64() % 100)

	if flag.RolloutPercent <= 0 {
		return true, false
	}
	if flag.RolloutPercent >= 100 {
		return true, true
	}
	return true, bucket < flag.RolloutPercent
}

func handleEvaluateFlag(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	user := r.URL.Query().Get("user")
	if user == "" {
		writeError(w, http.StatusBadRequest, "missing user")
		return
	}

	ok, enabled := store.Evaluate(key, user)
	if !ok {
		writeError(w, http.StatusNotFound, "flag not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"enabled": enabled})
}
