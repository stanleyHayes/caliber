package memory

import (
	"sync"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// keyedStore is a thread-safe in-memory aggregate store keyed by a primary id
// with one secondary index, preserving insertion order. It backs the dev
// repositories that share this shape (candidate, talent profile).
type keyedStore[T any] struct {
	mu        sync.RWMutex
	byID      map[kernel.ID]T
	bySecond  map[kernel.ID]kernel.ID
	order     []kernel.ID
	primary   func(*T) kernel.ID
	secondary func(*T) kernel.ID
}

func newKeyedStore[T any](primary, secondary func(*T) kernel.ID) *keyedStore[T] {
	return &keyedStore[T]{
		byID:      map[kernel.ID]T{},
		bySecond:  map[kernel.ID]kernel.ID{},
		primary:   primary,
		secondary: secondary,
	}
}

func (s *keyedStore[T]) create(v *T, dupMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.bySecond[s.secondary(v)]; exists {
		return kernel.Conflict(dupMsg)
	}
	id := s.primary(v)
	s.byID[id] = *v
	s.bySecond[s.secondary(v)] = id
	s.order = append(s.order, id)
	return nil
}

func (s *keyedStore[T]) get(id kernel.ID, notFound string) (*T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.byID[id]
	if !ok {
		return nil, kernel.NotFound(notFound)
	}
	return &v, nil
}

func (s *keyedStore[T]) getBySecondary(key kernel.ID, notFound string) (*T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.bySecond[key]
	if !ok {
		return nil, kernel.NotFound(notFound)
	}
	v := s.byID[id]
	return &v, nil
}

func (s *keyedStore[T]) update(v *T, notFound string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byID[s.primary(v)]; !ok {
		return kernel.NotFound(notFound)
	}
	s.byID[s.primary(v)] = *v
	return nil
}

// deleteByPrimary removes the record with the given primary id (and its secondary
// index + order entry). Deleting an absent id is a no-op, so erasure is idempotent.
func (s *keyedStore[T]) deleteByPrimary(id kernel.ID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.byID[id]
	if !ok {
		return
	}
	delete(s.bySecond, s.secondary(&v))
	delete(s.byID, id)
	for i, oid := range s.order {
		if oid == id {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
}

// deleteBySecondary removes the record indexed by the given secondary key.
func (s *keyedStore[T]) deleteBySecondary(key kernel.ID) {
	s.mu.RLock()
	id, ok := s.bySecond[key]
	s.mu.RUnlock()
	if ok {
		s.deleteByPrimary(id)
	}
}

func (s *keyedStore[T]) list(page kernel.Page) ([]*T, int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	total := int64(len(s.order))
	start := min(page.Offset(), len(s.order))
	end := min(start+page.Limit(), len(s.order))
	out := make([]*T, 0, end-start)
	for _, id := range s.order[start:end] {
		v := s.byID[id]
		out = append(out, &v)
	}
	return out, total
}

// reset removes every entry while keeping the store ready for reuse.
func (s *keyedStore[T]) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID = map[kernel.ID]T{}
	s.bySecond = map[kernel.ID]kernel.ID{}
	s.order = s.order[:0]
}
