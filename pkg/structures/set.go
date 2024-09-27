package structures

import (
	"encoding/json"
	"fmt"
)

type Set[T comparable] struct {
	data map[T]struct{}
}

func NewSet[T comparable]() *Set[T] {
	s := &Set[T]{}
	s.data = make(map[T]struct{})
	return s
}

func (s *Set[T]) AsSlice() []T {
	res := make([]T, len(s.data))
	i := 0
	for d := range s.data {
		res[i] = d
	}
	return res
}

func (s *Set[T]) Has(d T) bool {
	_, ok := s.data[d]
	return ok
}

func (s *Set[T]) Add(d T) {
	s.data[d] = struct{}{}
}

func (s *Set[T]) Remove(d T) {
	delete(s.data, d)
}

func (s *Set[T]) Clear() {
	s.data = make(map[T]struct{})
}

func (s *Set[T]) Size() int {
	return len(s.data)
}

func (s *Set[T]) AddM(list ...T) {
	for _, d := range list {
		s.Add(d)
	}
}

func (s *Set[T]) RemoveM(list ...T) {
	for _, d := range list {
		s.Remove(d)
	}
}

type SetFilterFunc[T comparable] func(d T) bool

func (s *Set[T]) Filter(P SetFilterFunc[T]) *Set[T] {
	res := NewSet[T]()
	for d := range s.data {
		if !P(d) {
			continue
		}
		res.Add(d)
	}
	return res
}

func (s *Set[T]) Union(s2 *Set[T]) *Set[T] {
	res := NewSet[T]()
	for d := range s.data {
		res.Add(d)
	}
	for d := range s2.data {
		res.Add(d)
	}
	return res
}

func (s *Set[T]) Intersect(s2 *Set[T]) *Set[T] {
	res := NewSet[T]()
	for d := range s.data {
		if !s2.Has(d) {
			continue
		}
		res.Add(d)
	}
	return res
}

func (s *Set[T]) Difference(s2 *Set[T]) *Set[T] {
	res := NewSet[T]()
	for d := range s.data {
		if s2.Has(d) {
			continue
		}
		res.Add(d)
	}
	return res
}

func (s *Set[T]) UnmarshalJSON(bytes []byte) error {
	var data []T
	err := json.Unmarshal(bytes, &data)
	if err != nil {
		return err
	}
	s.Clear()
	for _, d := range data {
		s.Add(d)
	}
	return nil
}

func (s *Set[T]) MarshalJSON() ([]byte, error) {
	data := s.AsSlice()
	return json.Marshal(data)
}

func (s Set[T]) String() string {
	return fmt.Sprintf("%v", s.AsSlice())
}
