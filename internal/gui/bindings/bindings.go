package bindings

import (
	"errors"

	"fyne.io/fyne/v2/data/binding"
	log "github.com/sirupsen/logrus"
)

type proxyBinding[B binding.DataItem] struct {
	from B
}

func (g *proxyBinding[B]) AddListener(listener binding.DataListener) {
	g.from.AddListener(listener)
}

func (g *proxyBinding[B]) RemoveListener(listener binding.DataListener) {
	g.from.RemoveListener(listener)
}

type proxyBindingMulti[B binding.DataItem] struct {
	data []B
}

func (g *proxyBindingMulti[B]) AddListener(listener binding.DataListener) {
	for _, d := range g.data {
		d.AddListener(listener)
	}
}

func (g *proxyBindingMulti[B]) RemoveListener(listener binding.DataListener) {
	for _, d := range g.data {
		d.RemoveListener(listener)
	}
}

type boundFloatMath struct {
	proxyBindingMulti[binding.Float]
	calc func(data ...float64) float64
}

func FloatMath(
	calc func(d ...float64) float64,
	data ...binding.Float,
) binding.Float {
	return &boundFloatMath{
		proxyBindingMulti: proxyBindingMulti[binding.Float]{data: data},
		calc:              calc,
	}
}

func (fm *boundFloatMath) Get() (float64, error) {
	var data []float64
	for _, d := range fm.data {
		v, err := d.Get()
		if err != nil {
			return 0.0, err
		}
		data = append(data, v)
	}
	val := fm.calc(data...)
	return val, nil
}

func (fm *boundFloatMath) Set(value float64) error {
	return nil
}

type Bound[T any] interface {
	binding.DataItem
	Get() (T, error)
	Set(T) error
}

func Listen[T any](data Bound[T], f func(T)) binding.DataListener {
	listener := binding.NewDataListener(func() {
		val, err := data.Get()
		if err != nil {
			log.Errorf("listen bind get error %s", err.Error())
			return
		}
		f(val)
	})
	data.AddListener(listener)
	return listener
}

func AddListenerToAll(f func(), items ...binding.DataItem) {

	listener := binding.NewDataListener(f)
	for _, item := range items {
		item.AddListener(listener)
	}
}

func ListenErr[T any](data Bound[T], f func(T), e func(error)) binding.DataListener {
	listener := binding.NewDataListener(func() {
		val, err := data.Get()
		if err != nil {
			e(err)
			return
		}
		f(val)
	})
	data.AddListener(listener)
	return listener
}

type boundMapping[F any, T any] struct {
	proxyBinding[Bound[F]]
	get func(F) (T, error)
	set func(T) (F, error)
}

func NewMapping[F any, T any](
	from Bound[F],
	f func(F) (T, error),
) Bound[T] {
	return &boundMapping[F, T]{
		proxyBinding: proxyBinding[Bound[F]]{from: from},
		get:          f,
	}
}

func NewReversableMapping[F any, T any](
	from Bound[F],
	get func(F) (T, error),
	set func(T) (F, error),
) Bound[T] {
	return &boundMapping[F, T]{
		proxyBinding: proxyBinding[Bound[F]]{from: from},
		get:          get,
		set:          set,
	}
}

func (bm *boundMapping[F, T]) Get() (T, error) {
	v, err := bm.from.Get()
	if err != nil {
		var t T
		log.Errorf("mapped bind get err %s", err.Error())
		return t, err
	}
	return bm.get(v)
}

func (bm *boundMapping[F, T]) Set(t T) error {
	if bm.set != nil {
		rev, err := bm.set(t)
		if err != nil {
			log.Errorf("mapped bind set err %s", err.Error())
			return err
		}
		return bm.from.Set(rev)
	}
	return nil
}

type ExternalBound[T any] interface {
	Bound[T]
	Reload() error
}

var errWrongType = errors.New("wrong type provided")

type mappedBinding[T any] struct {
	Bound[T]
	v T

	counter int
	self    binding.ExternalInt

	set func(T) error
	get func() (T, error)
}

func MappedBind[T any](get func() (T, error), set func(T) error) ExternalBound[T] {
	b := &mappedBinding[T]{
		get: get,
		set: set,
	}
	val, err := b.get()
	if err == nil {
		b.v = val
	}
	b.self = binding.BindInt(&b.counter)
	return b
}

func (m *mappedBinding[T]) AddListener(listener binding.DataListener) {
	m.self.AddListener(listener)
}

func (m *mappedBinding[T]) RemoveListener(listener binding.DataListener) {
	m.self.RemoveListener(listener)
}

func (m *mappedBinding[T]) Reload() error {
	val, err := m.get()
	if err != nil {
		return err
	}
	m.v = val
	m.self.Set(m.counter + 1)
	return nil
}

func (m *mappedBinding[T]) Get() (T, error) {
	val, err := m.get()
	if err != nil {
		var t T
		return t, err
	}
	m.v = val
	return val, nil
}

func (m *mappedBinding[T]) Set(val T) error {
	err := m.set(val)
	if err != nil {
		return err
	}
	m.v = val
	m.self.Set(m.counter + 1)
	return nil
}

type ExternalBoundList[T any] interface {
	ExternalBound[[]T]
	GetItem(index int) (binding.DataItem, error)
	Length() int
}

var errOutOfBounds = errors.New("index out of bounds")

type mappedListBinding[T any] struct {
	mappedBinding[[]T]
}

func MappedListBind[T any](get func() ([]T, error), set func([]T) error) ExternalBoundList[T] {
	b := &mappedListBinding[T]{
		mappedBinding: mappedBinding[[]T]{
			get: get,
			set: set,
		},
	}
	val, err := b.get()
	if err == nil {
		b.v = val
	}
	b.self = binding.BindInt(&b.counter)
	return b
}

func (ml *mappedListBinding[T]) GetItem(index int) (binding.DataItem, error) {
	if index < 0 || index >= len(ml.v) {
		return nil, errOutOfBounds
	}
	return NewReversableMapping(
		ml,
		func(l []T) (T, error) {
			if index < 0 || index >= len(l) {
				var t T
				return t, errOutOfBounds
			}
			return l[index], nil
		},
		func(val T) ([]T, error) {
			l, err := ml.Get()
			if err != nil {
				return nil, err
			}
			if index < 0 || index >= len(l) {
				return nil, errOutOfBounds
			}
			l[index] = val
			return l, nil
		},
	), nil
}

func (ml *mappedListBinding[T]) Length() int {
	return len(ml.v)
}
