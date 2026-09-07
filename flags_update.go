package main

import (
	"encoding/json"
	"net/http"
)

type FlagPatch struct {
	Enabled        *bool   `json:"enabled"`
	Description    *string `json:"description"`
	RolloutPercent *int    `json:"rollout_percent"`
}

func (s *Store) Update(key string, p FlagPatch) (Flag, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	flag, ok := s.flags[key]
	if !ok {
		return Flag{}, false
	}

	if p.Enabled != nil {
		flag.Enabled = *p.Enabled
	}
	if p.Description != nil {
		flag.Description = *p.Description
	}
	if p.RolloutPercent != nil {
		flag.RolloutPercent = *p.RolloutPercent
	}

	s.flags[key] = flag
	return flag, true
}

func handleUpdateFlag(w http.ResponseWriter, r *http.Request) {
	var p FlagPatch
	if err := DecodeJSON(w, r, &p); err != nil {
		return
	}

	if p.RolloutPercent != nil && (*p.RolloutPercent < 0 || *p.RolloutPercent > 100) {
		writeError(w, http.StatusBadRequest, "rollout_percent must be between 0 and 100")
		return
	}

	key := r.PathValue("key")
	flag, ok := store.Update(key, p)
	if !ok {
		writeError(w, http.StatusNotFound, "flag not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(flag)
}
