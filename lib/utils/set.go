package utils

import (
	"sync"

	"golang.org/x/exp/constraints"
)

type Set[T constraints.Ordered] struct {
	mu  *sync.Mutex
	set []T
}

func (s *Set[T]) Len() int {
	return len(s.set)
}

func (s *Set[T]) Range() []T {
	return s.set
}

func (s *Set[T]) Set(v T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, i := range s.set {
		if i == v {
			return
		}
	}

	s.set = append(s.set, v)
}

func (s *Set[T]) UnSet(v T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var result []T
	for _, i := range s.set {
		if i != v {
			result = append(result, i)
		}
	}

	s.set = result
}

func SetInt64() *Set[int64] {
	return &Set[int64]{
		mu: new(sync.Mutex),
	}
}
