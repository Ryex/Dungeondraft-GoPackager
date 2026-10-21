package named

import (
	"sync"

	"fyne.io/fyne/v2"
)

// Cache resolves the names of the parts of a user interface to the objects
// they belong to.
//
// The names are looked up in the object tree of the interface, so that a name
// stays valid when the interface is rebuilt: the object behind the name
// changes, the name does not.
type Cache struct {
	mu         sync.RWMutex
	items      map[string]fyne.CanvasObject
	generation uint64

	root func() fyne.CanvasObject
}

// NewCache returns a cache that looks the names up in the object tree below
// the object returned by root.
func NewCache(root func() fyne.CanvasObject) *Cache {
	return &Cache{
		items: map[string]fyne.CanvasObject{},
		root:  root,
	}
}

// Get returns the object with the given name.
//
// It returns nil when there is no object with that name in the interface.
func (c *Cache) Get(name string) fyne.CanvasObject {
	c.mu.RLock()
	obj, ok := c.items[name]
	generation := c.generation
	c.mu.RUnlock()
	if ok {
		return obj
	}

	root := c.root()
	if root == nil {
		return nil
	}

	found := find(root, name)
	if found == nil {
		return nil
	}

	c.mu.Lock()
	// the tree may have changed while it was walked, in which case the
	// result is not worth remembering
	if c.generation == generation {
		c.items[name] = found
	}
	c.mu.Unlock()

	return found
}

// Reset forgets all the cached objects, e.g. after the interface was rebuilt.
func (c *Cache) Reset() {
	c.mu.Lock()
	c.generation++
	c.items = map[string]fyne.CanvasObject{}
	c.mu.Unlock()
}

// find searches the object tree below root for the object with the given name.
func find(root fyne.CanvasObject, name string) fyne.CanvasObject {
	queue := []fyne.CanvasObject{root}
	for len(queue) > 0 {
		obj := queue[0]
		queue = queue[1:]

		if named, ok := obj.(*Named); ok {
			if named.name == name {
				return named.inner
			}
			queue = append(queue, named.inner)
			continue
		}

		queue = append(queue, Children(obj)...)
	}
	return nil
}
