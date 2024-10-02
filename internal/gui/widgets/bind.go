package widgets

import (
	"sync"
	"sync/atomic"

	"fyne.io/fyne/v2/data/binding"
)

type listenerPair struct {
	data     binding.DataItem
	listener binding.DataListener
}

type binder struct {
	callback atomic.Pointer[func(binding.DataItem)]
	lock     sync.RWMutex
	pair     listenerPair // guarded by lock
}

func (b *binder) Bind(data binding.DataItem) {
	listener := binding.NewDataListener(func() {
		f := b.callback.Load()
		if f == nil || *f == nil {
			return
		}
		(*f)(data)
	})
	data.AddListener(listener)
	pair := listenerPair{
	  data: data,
	  listener: listener,
	}
	b.lock.Lock()
	b.unbindLocked()
	b.pair =pair
	b.lock.Unlock()
}

func (b *binder) CallWithData(f func(data binding.DataItem)) {
  b.lock.RLock()
  data := b.pair.data
  b.lock.RUnlock()
  f(data)
}

func (b *binder) SetCallback(f func(data binding.DataItem)) {
  b.callback.Store(&f)
}

func (b *binder) Unbind() {
  b.lock.Lock()
  b.unbindLocked()
  b.lock.Unlock()
}

func(b *binder) unbindLocked() {
  prev := b.pair
  b.pair = listenerPair{nil, nil}
  if prev.listener == nil || prev.data == nil {
    return
  }
  prev.data.RemoveListener(prev.listener)
}
