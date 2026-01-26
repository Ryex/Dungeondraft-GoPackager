package structures

import (
	"errors"
)

var ErrStackEmpty = errors.New("stack empty")

type Stack[T any] struct {
	data []T
}

func NewStack[T any]() *Stack[T] {
	s := &Stack[T]{ make([]T, 0)}
	return s
}

func (s *Stack[T]) Push(x T) {
	s.data = append(s.data, x)
}

func (s *Stack[T]) Pop() (T, error) {
	if len(s.data) == 0 {
		var empty T
		return empty, ErrStackEmpty
	}
	item := s.data[len(s.data)-1]
	s.data = s.data[:len(s.data)-1]
	return item, nil
}

func (s *Stack[T]) Peek() (*T, error) {
	if len(s.data) == 0 {
		return nil, ErrStackEmpty
	}
	item := &s.data[len(s.data)-1]
	return item, nil
}

func (s *Stack[T]) AsSlice() []T {
	return s.data
}

func (s *Stack[T]) Size() int {
	return len(s.data)
}

func (s *Stack[T]) IsEmpty() bool {
	return len(s.data) > 0
}


