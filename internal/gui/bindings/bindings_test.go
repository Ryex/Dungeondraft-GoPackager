package bindings

import (
	"errors"
	"os"
	"strings"
	"testing"

	"fyne.io/fyne/v2/data/binding"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ryex/dungeondraft-gopackager/internal/testutil"
)

func TestMain(m *testing.M) {
	testutil.SetupFyneApp()
	os.Exit(m.Run())
}

// failingBound is a bound item whose reads and writes always fail. It is used
// to exercise the error handling of the helpers in this package.
type failingBound struct{}

func (failingBound) Get() (string, error)                { return "", errors.New("get failed") }
func (failingBound) Set(string) error                    { return errors.New("set failed") }
func (failingBound) AddListener(binding.DataListener)    {}
func (failingBound) RemoveListener(binding.DataListener) {}

func TestListenCallsBackWithCurrentValue(t *testing.T) {
	str := binding.NewString()
	require.NoError(t, str.Set("hello"))

	var got []string
	listener := Listen(str, func(v string) {
		got = append(got, v)
	})
	defer str.RemoveListener(listener)

	// the listener is called with the current value as soon as it is attached
	assert.Equal(t, []string{"hello"}, got)

	require.NoError(t, str.Set("world"))
	assert.Equal(t, []string{"hello", "world"}, got)
}

func TestListenReportsErrors(t *testing.T) {
	var called bool
	listener := Listen(failingBound{}, func(string) {
		called = true
	})

	// the value can not be read, so the callback must not run
	listener.DataChanged()

	assert.False(t, called, "the callback must not run when the value can not be read")
}

func TestListenErrCallsErrorHandler(t *testing.T) {
	var gotErr error
	var called bool
	listener := ListenErr(failingBound{}, func(string) {
		called = true
	}, func(err error) {
		gotErr = err
	})

	listener.DataChanged()

	assert.False(t, called, "the value callback must not run on error")
	require.Error(t, gotErr)
	assert.Equal(t, "get failed", gotErr.Error())
}

func TestNewReversibleMappingRoundTrip(t *testing.T) {
	str := binding.NewString()
	require.NoError(t, str.Set("en"))

	mapped := NewReversibleMapping(
		str,
		func(v string) (string, error) { return strings.ToUpper(v), nil },
		func(v string) (string, error) { return strings.ToLower(v), nil },
	)

	got, err := mapped.Get()
	require.NoError(t, err)
	assert.Equal(t, "EN", got)

	require.NoError(t, mapped.Set("Mixed"))

	back, err := str.Get()
	require.NoError(t, err)
	assert.Equal(t, "mixed", back, "setting the mapping must write back through the reverse function")
}

func TestNewMappingHasNoWriteBack(t *testing.T) {
	str := binding.NewString()
	require.NoError(t, str.Set("en"))

	mapped := NewMapping(str, func(v string) (string, error) {
		return strings.ToUpper(v), nil
	})

	got, err := mapped.Get()
	require.NoError(t, err)
	assert.Equal(t, "EN", got)

	require.NoError(t, mapped.Set("ignored"))

	back, err := str.Get()
	require.NoError(t, err)
	assert.Equal(t, "en", back, "a one way mapping must not write back")
}

func TestMappedBindReloadTriggersListener(t *testing.T) {
	value := 1
	bound := MappedBind(
		func() (int, error) { return value, nil },
		func(v int) error { value = v; return nil },
	)

	calls := 0
	bound.AddListener(binding.NewDataListener(func() {
		calls++
	}))

	// attaching a listener immediately reports the current value
	assert.Equal(t, 1, calls)

	value = 2
	require.NoError(t, bound.Reload())
	assert.Equal(t, 2, calls, "reloading must notify listeners")

	got, err := bound.Get()
	require.NoError(t, err)
	assert.Equal(t, 2, got)
}

func TestMappedBindSetWritesThrough(t *testing.T) {
	value := 1
	bound := MappedBind(
		func() (int, error) { return value, nil },
		func(v int) error { value = v; return nil },
	)

	require.NoError(t, bound.Set(42))
	assert.Equal(t, 42, value)
}

func TestFloatMathCombinesBindings(t *testing.T) {
	a := binding.NewFloat()
	b := binding.NewFloat()
	require.NoError(t, a.Set(2))
	require.NoError(t, b.Set(3))

	sum := FloatMath(func(values ...float64) float64 {
		total := 0.0
		for _, v := range values {
			total += v
		}
		return total
	}, a, b)

	got, err := sum.Get()
	require.NoError(t, err)
	assert.Equal(t, 5.0, got)

	require.NoError(t, a.Set(10))
	got, err = sum.Get()
	require.NoError(t, err)
	assert.Equal(t, 13.0, got)
}

func TestMappedListBindGetItem(t *testing.T) {
	list := []string{"a", "b"}
	bound := MappedListBind(
		func() ([]string, error) { return list, nil },
		func(v []string) error { list = v; return nil },
	)

	assert.Equal(t, 2, bound.Length())

	item, err := bound.GetItem(1)
	require.NoError(t, err)

	typed, ok := item.(Bound[string])
	require.True(t, ok, "GetItem must return a Bound value")

	got, err := typed.Get()
	require.NoError(t, err)
	assert.Equal(t, "b", got)

	require.NoError(t, typed.Set("c"))

	updated, err := bound.Get()
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "c"}, updated)
}

func TestMappedListBindGetItemOutOfBounds(t *testing.T) {
	list := []string{"a"}
	bound := MappedListBind(
		func() ([]string, error) { return list, nil },
		func(v []string) error { list = v; return nil },
	)

	_, err := bound.GetItem(5)
	assert.ErrorIs(t, err, errOutOfBounds)
}
