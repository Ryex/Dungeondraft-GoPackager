package custom_binding

import "fyne.io/fyne/v2/data/binding"


type boundFloats struct {
	data []binding.Float
}

func (g *boundFloats) AddListener(listener binding.DataListener) {
	for _, d := range g.data {
		d.AddListener(listener)
	}
}

func (g *boundFloats) RemoveListener(listener binding.DataListener) {
	for _, d := range g.data {
		d.RemoveListener(listener)
	}
}

type boundFloatMath struct {
	boundFloats
	calc func(data ...float64) float64
}

func FloatMath(
	calc func(d ...float64) float64,
	data ...binding.Float,
) binding.Float {
	return &boundFloatMath{
		boundFloats: boundFloats{data: data},
		calc:        calc,
	}
}

func (fm *boundFloatMath)Get() (float64, error) {
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
