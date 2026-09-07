package main

import (
	"errors"
	"sync"
)

type Flag struct {
	Key            string `json:"key"`
	Enabled        bool   `json:"enabled"`
	Description    string `json:"description,omitempty"`
	RolloutPercent int    `json:"rollout_percent"`
}

var (
	ErrNotFound = errors.New("flag not found")
	ErrConflict = errors.New("flag already exists")
)

type Store struct {
	mu    sync.RWMutex
	flags map[string]Flag
}

func NewStore() *Store {
	return &Store{flags: make(map[string]Flag)}
}

func (s *Store) Healthy() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.flags != nil
}
