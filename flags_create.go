package main

import (
	"encoding/json"
	"errors"
	"net/http"
)

type createFlagRequest struct {
	Key            string `json:"key"`
	Enabled        *bool  `json:"enabled"`
	Description    string `json:"description"`
	RolloutPercent *int   `json:"rollout_percent"`
}

func (s *Store) Create(f Flag) (Flag, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.flags[f.Key]; exists {
		return Flag{}, ErrConflict
	}
	s.flags[f.Key] = f
	return f, nil
}

func handleCreateFlag(w http.ResponseWriter, r *http.Request) {
	var req createFlagRequest
	if err := DecodeJSON(w, r, &req); err != nil {
		return
	}

	if req.Key == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}
	if req.Enabled == nil {
		writeError(w, http.StatusBadRequest, "enabled is required")
		return
	}

	rollout := 0
	if req.RolloutPercent != nil {
		if *req.RolloutPercent < 0 || *req.RolloutPercent > 100 {
			writeError(w, http.StatusBadRequest, "rollout_percent must be between 0 and 100")
			return
		}
		rollout = *req.RolloutPercent
	}

	flag := Flag{
		Key:            req.Key,
		Enabled:        *req.Enabled,
		Description:    req.Description,
		RolloutPercent: rollout,
	}

	created, err := store.Create(flag)
	if err != nil {
		if errors.Is(err, ErrConflict) {
			writeError(w, http.StatusConflict, "flag already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}
