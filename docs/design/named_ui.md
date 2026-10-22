# Named parts of the user interface

This is implemented: the parts of the interface are named where they are
created, and looked up by name afterwards.

## The problem

The parts of the user interface are addressed by name. Today they are kept in
`uiRegions` (`internal/gui/regions.go`), a struct with one field per part of
the interface:

```go
a.ui.Header = welcome
a.ui.ThemeToggle = darkThemeToggle
```

Every named part has to be declared in the struct, and every place that creates
it has to remember to assign it. A new part of the interface therefore means
three edits: the declaration, the assignment and — for the parts the
specification tests address — the test.

## The idea

Name the part where it is created, and look it up by name afterwards.

```go
// the name travels with the object
welcome := named.New("header", container.NewPadded(...))

// the part is found in the object tree of the window
header := a.ui.Get("header")
```

`named.Named` is a transparent wrapper around a canvas object. It keeps the
name, and it does not change the way the object is rendered, laid out or
interacted with. `named.Cache` resolves a name to the object it belongs to by
walking the object tree of the window, and remembers the result.

## What Fyne allows

Fyne walks the object tree with two type switches
(`internal/driver/util.go`, `walkObjectTree`):

```go
switch co := obj.(type) {
case *fyne.Container:
	children = co.Objects
case fyne.Widget:
	renderer = cache.Renderer(co)
	children = renderer.Objects()
}
```

A wrapper that is neither a `*fyne.Container` nor a `fyne.Widget` is a leaf
for Fyne: its children are never painted, never receive mouse events and are
never laid out. A wrapper therefore has to be a `fyne.Widget`.

A widget wrapper that renders the wrapped object as its only child keeps
everything working:

- the wrapped object stays in the object tree, so it is painted, it receives
  the mouse and keyboard events and it can be focused;
- the theme is applied to it, because `app.ApplyThemeTo` walks `fyne.Widget`s
  through their renderer;
- `canvas.Refresh` reaches it, because the driver registers every object of the
  tree with its canvas.

The wrapped object keeps its own position inside the wrapper, so the geometry
of the tree does not change: the wrapper is positioned by the layout of its
parent, the wrapped object stays at the origin of the wrapper.

### The renderer has to be registered

`driver.WalkCompleteObjectTree` — the walk the focus manager uses to check
that an object may take the focus — only descends into a widget that has a
renderer in the render cache:

```go
case fyne.Widget:
	if requireVisible || cache.IsRendered(co) {
		renderer = cache.Renderer(co)
		children = renderer.Objects()
	}
```

A wrapper that does not have a renderer yet is invisible to that walk, and the
objects below it cannot be focused. Embedding `widget.BaseWidget` fixes this:
`BaseWidget.MinSize` asks the render cache for the renderer of the wrapper, so
the wrapper is registered with the first layout pass — long before the user
can interact with it.

## The API

```go
package named

// Named wraps a canvas object and carries a name with it.
func New(name string, obj fyne.CanvasObject) *Named

func (n *Named) Name() string
func (n *Named) Object() fyne.CanvasObject

// Cache resolves the names of the parts of a user interface.
func NewCache(root func() fyne.CanvasObject) *Cache

// Get returns the object with the given name, or nil if there is none.
func (c *Cache) Get(name string) fyne.CanvasObject

// Reset forgets the cached objects, e.g. after the interface was rebuilt.
func (c *Cache) Reset()
```

`Get` returns the *wrapped* object, so that the caller gets the widget it
expects and not the wrapper:

```go
toggle, ok := a.ui.Get("themeToggle").(*widgets.Toggle)
```

## The cache and the lifetime of the interface

The name is stable, the object behind it is not: the interface is rebuilt on
every locale change and the button panel is replaced on every pack that is
loaded. The tree is therefore the source of truth and the cache is only a
memoization of the last lookup.

The cache has to be invalidated whenever the tree changes:

- `buildMainUI` creates a new cache, so nothing has to be done there;
- `setMainContent` replaces the content of the window, so it has to reset the
  cache;
- the selection of the resource tree replaces the information pane, so it has
  to reset the cache as well.

Because `setMainContent` runs on the Fyne goroutine and a lookup can happen
on another one, the cache needs a mutex, and a lookup that is running while
the tree changes must not store its result. A generation counter solves that:

```go
func (c *Cache) Get(name string) fyne.CanvasObject {
	// ... read the cache ...
	generation := c.generation
	// ... walk the tree ...
	c.mu.Lock()
	if c.generation == generation {
		c.items[name] = found
	}
	c.mu.Unlock()
	return found
}
```

The old `uiRegions` struct had the same problem in a worse form: `ButtonPanel`
and `SplitPane` were written from the goroutine that loads the package and
read from the main thread, without a lock. The cache with a mutex fixes that
race on the way.

## The specification tests

The DSL of `internal/testutil/uitest` addresses the parts of the interface by
name. With the wrapper it needs to know about it in two places:

- `children` has to look through the wrapper, otherwise the objects below it
  are invisible to every selector;
- a `Named(name)` selector resolves a name in the scope of the expectation, so
  that the expectation only holds when the named part is part of the scope.

```go
uitest.Expect(t, uitest.In(namedPart(t, a, nameHeader)),
	uitest.Has("a theme toggle on the left", uitest.Named(nameThemeToggle)).LeftOfCenter(),
	uitest.Has("a language select dropdown to the right of the theme toggle",
		uitest.Named(nameLanguageSelect)).RightOf(uitest.Named(nameThemeToggle)),
)
```

The scope of an expectation is the object the name resolves to, so the
relations keep working across the wrapper: the wrapped object is at the origin
of the wrapper, the wrapper is at the position of the part in the interface.

`internal/gui/gui_test.go` had its own copy of the traversal of the object
tree; it uses `named.Children` now, so that the test traversal and the lookup
traversal cannot drift apart.

## The names

The names are part of the interface between the application and the tests, so
they are constants. They are defined in the file that creates the part, next
to the code that creates it:

```go
// internal/gui/gui.go
const (
	nameHeader = "header"
	// ...
)
```

A constant turns a typo into a compile error, which is what the fields of the
old `uiRegions` struct did.

| Name             | Constant              | Created in          |
| ---------------- | --------------------- | ------------------- |
| `mainColumn`     | `nameMainColumn`      | `gui.go`            |
| `header`         | `nameHeader`          | `gui.go`            |
| `themeToggle`    | `nameThemeToggle`     | `gui.go`            |
| `languageSelect` | `nameLanguageSelect`  | `gui.go`            |
| `titleText`      | `nameTitleText`       | `gui.go`            |
| `subTitleText`   | `nameSubTitleText`    | `gui.go`            |
| `aboutButton`    | `nameAboutButton`     | `gui.go`            |
| `githubButton`   | `nameGithubButton`    | `gui.go`            |
| `infoButton`     | `nameInfoButton`      | `gui.go`            |
| `pathSection`    | `namePathSection`     | `gui.go`            |
| `pathEntry`      | `namePathEntry`       | `gui.go`            |
| `openPackButton` | `nameOpenPackButton`  | `gui.go`            |
| `openFolderButton` | `nameOpenFolderButton` | `gui.go`          |
| `content`        | `nameContent`         | `gui.go`            |
| `packPanel`      | `namePackPanel`       | `pack.go`           |
| `extractPanel`   | `nameExtractPanel`    | `unpack.go`         |
| `resourceTree`   | `nameResourceTree`    | `package.go`        |
| `resourcePane`   | `nameResourcePane`    | `package.go`        |
| `splitPane`      | `nameSplitPane`       | `package.go`        |
| `filterEntry`    | `nameFilterEntry`     | `package.go`        |
| `displayByRadio` | `nameDisplayByRadio`  | `package.go`        |
| `infoTabs`       | `nameInfoTabs`        | `package.go`        |
| `filePreview`    | `nameFilePreview`     | `package.go`        |
| `tagInfo`        | `nameTagInfo`         | `package.go`        |
| `tagList`        | `nameTagList`         | `package.go`        |
| `metadataPane`   | `nameMetadataPane`    | `package.go`        |
| `metadataForm`   | `nameMetadataForm`    | `package.go`        |
| `bulkOpsPane`    | `nameBulkOpsPane`     | `package.go`        |
| `bulkOpsTabs`    | `nameBulkOpsTabs`     | `package.go`        |
| `bulkTagHeader`  | `nameBulkTagHeader`   | `package.go`        |
| `bulkResourceList` | `nameBulkResourceList` | `package.go`      |
| `bulkTagList`    | `nameBulkTagList`     | `package.go`        |
| `bulkTagSelector` | `nameBulkTagSelector` | `package.go`        |
| `bulkTagAddButton` | `nameBulkTagAddButton` | `package.go`      |
| `thumbnailToggle` | `nameThumbnailToggle` | `package.go`        |
| `imagePreview`   | `nameImagePreview`    | `package.go`        |
| `textPreview`    | `nameTextPreview`     | `package.go`        |
| `tooLargeLabel`  | `nameTooLargeLabel`   | `package.go`        |

The two button panels have different names, because they are different parts
of the interface: the one of `pack.go` packs an unpacked folder, the one of
`unpack.go` extracts a packed file. The split pane, the resource tree and the
resource pane are the same for both, so they share their names.

## The trade-offs

**For**

- the name of a part is declared where the part is created;
- the specification tests read like the specification;
- no central struct to keep in sync with the interface;
- the race on `uiRegions` disappears.

**Against**

- a name is a string: a typo is a runtime error, not a compile error (mitigated
  by constants);
- `Get` returns a `fyne.CanvasObject`, so the caller has to assert the type
  (the fields of `uiRegions` are typed);
- every named part adds a node to the object tree;
- the wrapper has to be transparent, which means it has to follow the rules of
  the Fyne internals — the renderer registration above is the subtle part of
  it.

## The alternative

The name could also be kept in a side table instead of in the object tree:

```go
func Named(name string, obj fyne.CanvasObject) fyne.CanvasObject
```

That leaves the object tree completely untouched — no extra node, no renderer
registration, no focus subtleties — at the price of a global map from objects
to names that has to be cleaned up when the interface is rebuilt. The wrapper
is the better trade as long as it stays transparent, which the tests of
`internal/gui/named` show.
