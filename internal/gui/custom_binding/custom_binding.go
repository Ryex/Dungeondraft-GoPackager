package custom_binding

import (
	"fyne.io/fyne/v2/data/binding"
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

type boundMapping[F any, T any] struct {
	proxyBinding[Bound[F]]
	f func(F) (T, error)
	r func(T) (F, error)
}

func NewMapping[F any, T any](
	from Bound[F],
	f func(F) (T, error),
) Bound[T] {
	return &boundMapping[F, T]{
		proxyBinding: proxyBinding[Bound[F]]{from: from},
		f: f,
	}
}
func NewReversableMapping[F any, T any](
	from Bound[F],
	f func(F) (T, error),
	r func(T) (F, error),
) Bound[T] {
	return &boundMapping[F, T]{
		proxyBinding: proxyBinding[Bound[F]]{from: from},
		f: f,
		r: r,
	}
}

func(bm *boundMapping[F, T]) Get() (T, error) {
	v, err := bm.from.Get()
	if err != nil {
		var t T
		return t, err
	}
	return bm.f(v)
}

func(bm *boundMapping[F, T]) Set(t T) error {
	if bm.r != nil {
		rev, err := bm.r(t)
		if err != nil {
			return err
		}
		return bm.from.Set(rev)
	}
	return nil
}

var test = binding.NewStringTree()
