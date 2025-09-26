package repository

import (
	"bytes"
	"sync"
)

type Store struct {
	mu      sync.Mutex
	history map[string]*bytes.Buffer
}

func NewStore() *Store {
	return &Store{history: make(map[string]*bytes.Buffer)}
}

func (s *Store) AppendBatch(jid string, lines []string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.history[jid]; !ok {
		s.history[jid] = &bytes.Buffer{}
	}
	for _, ln := range lines {
		s.history[jid].WriteString(ln)
		if len(ln) == 0 || ln[len(ln)-1] != '\n' {
			s.history[jid].WriteByte('\n')
		}
	}
	return s.history[jid].Bytes(), nil
}

func (s *Store) Bytes(jid string) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	if buf, ok := s.history[jid]; ok {
		return buf.Bytes()
	}
	return nil
}
