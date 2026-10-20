package structures

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStackPushAndPop(t *testing.T) {
	stack := NewStack[string]()

	assert.True(t, stack.IsEmpty())
	assert.Equal(t, 0, stack.Size())

	stack.Push("a", "b")
	assert.False(t, stack.IsEmpty())
	assert.Equal(t, 2, stack.Size())

	// the last pushed item comes out first
	item, err := stack.Pop()
	require.NoError(t, err)
	assert.Equal(t, "b", item)

	item, err = stack.Pop()
	require.NoError(t, err)
	assert.Equal(t, "a", item)

	assert.True(t, stack.IsEmpty())

	_, err = stack.Pop()
	assert.ErrorIs(t, err, ErrStackEmpty)
}

func TestStackPeek(t *testing.T) {
	stack := NewStack[int]()
	stack.Push(1, 2)

	item, err := stack.Peek()
	require.NoError(t, err)
	assert.Equal(t, 2, *item)

	// peeking must not remove the item
	assert.Equal(t, 2, stack.Size())
}

func TestNewStackFromSlice(t *testing.T) {
	stack := NewStackFromSlice([]string{"a", "b", "c"})

	assert.False(t, stack.IsEmpty())
	assert.Equal(t, 3, stack.Size())
	assert.Equal(t, []string{"a", "b", "c"}, stack.AsSlice())

	item, err := stack.Pop()
	require.NoError(t, err)
	assert.Equal(t, "c", item)
}
