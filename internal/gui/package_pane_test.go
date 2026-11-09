package gui

import (
	"testing"

	"fyne.io/fyne/v2"
)

// resourcePaneContent returns the object the resource pane shows right now.
func resourcePaneContent(t *testing.T, a *App) fyne.CanvasObject {
	t.Helper()

	pane, ok := a.ui.Get(nameResourcePane).(*fyne.Container)
	if !ok {
		t.Fatal("the resource pane is not a stack")
	}
	if len(pane.Objects) == 0 {
		t.Fatal("the resource pane shows nothing")
	}
	return pane.Objects[0]
}

// TestBulkPaneIsRebuiltOnlyWhenTheKindChanges verifies that the resource pane
// is rebuilt when the kind of the content it shows changes, and that a change
// of the selection within the bulk operations only updates the pane in place.
func TestBulkPaneIsRebuiltOnlyWhenTheKindChanges(t *testing.T) {
	a := newSpecApp(t)
	dir := packFixtureDir(t)

	a.loadUnpackedPath(dir)
	waitForPackLoad(t, a)

	// the hashes are computed up front, so that no background work touches
	// the interface while it is inspected
	computeMd5s(t, a.pkg)

	tree := resourceTree(t, a)

	// one resource shows its information
	tree.Select(resPath("textures/objects/Rock.png"))
	waitForBackgroundWork(t, a)

	single := resourcePaneContent(t, a)

	// a second resource shows the bulk operations, which replaces the
	// information of the single resource
	tree.Select(resPath("textures/objects/Tree.png"))
	waitForBackgroundWork(t, a)

	bulk := resourcePaneContent(t, a)
	if single == bulk {
		t.Fatal("the bulk operations pane must replace the information of the single resource")
	}
	if a.ui.Get(nameBulkOpsPane) == nil {
		t.Fatal("the bulk operations pane must be shown")
	}

	// a third resource keeps the bulk operations, they are only updated
	// with the new selection
	tree.Select(resPath("textures/walls/Stone.png"))
	waitForBackgroundWork(t, a)

	if grown := resourcePaneContent(t, a); grown != bulk {
		t.Fatal("the bulk operations pane must be updated in place, not rebuilt")
	}
}
