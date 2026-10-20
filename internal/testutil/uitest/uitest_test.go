package uitest

import (
	"os"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	test.NewApp()
	os.Exit(m.Run())
}

// tree builds a deterministic widget tree:
//
//	root (200x100)
//	├── top    label   "Title"       (0,0)   200x20
//	├── left   button  "Open pack"   (0,30)   40x20
//	├── right  button  "Open folder" (120,30) 40x20
//	└── middle button  "Middle"      (80,60)  40x20
//
// The "middle" button is centered horizontally in the root, "left" is left of
// the center and "right" is right of it. "middle" is the bottommost widget.
func tree() (root *fyne.Container, top *widget.Label, left, right, middle *widget.Button) {
	top = widget.NewLabel("Title")
	left = widget.NewButton("Open pack", nil)
	right = widget.NewButton("Open folder", nil)
	middle = widget.NewButton("Middle", nil)

	top.Move(fyne.NewPos(0, 0))
	top.Resize(fyne.NewSize(200, 20))
	left.Move(fyne.NewPos(0, 30))
	left.Resize(fyne.NewSize(40, 20))
	right.Move(fyne.NewPos(120, 30))
	right.Resize(fyne.NewSize(40, 20))
	middle.Move(fyne.NewPos(80, 60))
	middle.Resize(fyne.NewSize(40, 20))

	root = container.NewWithoutLayout(top, left, right, middle)
	root.Resize(fyne.NewSize(200, 100))
	return root, top, left, right, middle
}

func TestHasFindsTheWidget(t *testing.T) {
	root, _, _, _, _ := tree()

	failures := Has("an open pack button", Button("Open pack")).Check(In(root))
	assert.Empty(t, failures)
}

func TestHasFailsWhenTheWidgetIsMissing(t *testing.T) {
	root, _, _, _, _ := tree()

	failures := Has("a close button", Button("Close")).Check(In(root))
	require.Len(t, failures, 1)
	assert.Equal(t, []string{"a close button"}, failures[0].Path)
	assert.Contains(t, failures[0].Message, "found none")
	assert.Contains(t, strings.Join(failures[0].Details, "\n"), "Open pack")
}

func TestHasFailsWhenTheWidgetIsAmbiguous(t *testing.T) {
	first := widget.NewButton("Same", nil)
	second := widget.NewButton("Same", nil)
	root := container.NewWithoutLayout(first, second)
	root.Resize(fyne.NewSize(100, 100))

	failures := Has("a button", Button("Same")).Check(In(root))
	require.Len(t, failures, 1)
	assert.Contains(t, failures[0].Message, "found 2")
}

func TestHasAnyAcceptsSeveralMatches(t *testing.T) {
	root, _, _, _, _ := tree()

	failures := HasAny("an open button", Any(
		Button("Open pack"),
		Button("Open folder"),
	)).Check(In(root))
	assert.Empty(t, failures)
}

func TestNestedExpectationsUseTheScopeOfTheMatch(t *testing.T) {
	inner := container.NewWithoutLayout(widget.NewButton("Inner", nil))
	inner.Move(fyne.NewPos(0, 0))
	inner.Resize(fyne.NewSize(100, 50))
	outer := container.NewWithoutLayout(inner, widget.NewButton("Outer", nil))
	outer.Resize(fyne.NewSize(200, 100))

	failures := Has("an inner container", Object(inner),
		Has("an inner button", Button("Inner")),
	).Check(In(outer))
	assert.Empty(t, failures)
}

func TestNestedExpectationsFallBackToTheParentScope(t *testing.T) {
	// the subject is a button, which holds no children, so the nested
	// expectation has to be resolved in the parent scope
	root, _, left, _, _ := tree()

	failures := Has("an open pack button", Object(left),
		Has("an open folder button", Button("Open folder")),
	).Check(In(root))
	assert.Empty(t, failures)
}

func TestRightOfRelation(t *testing.T) {
	root, _, left, right, _ := tree()

	assert.Empty(t, Has("a button", Object(right)).
		RightOf(Object(left)).Check(In(root)))

	failures := Has("a button", Object(left)).
		RightOf(Object(right)).Check(In(root))
	require.Len(t, failures, 1)
	assert.Contains(t, failures[0].Message, "to the right of")
}

func TestLeftOfRelation(t *testing.T) {
	root, _, left, right, _ := tree()

	assert.Empty(t, Has("a button", Object(left)).
		LeftOf(Object(right)).Check(In(root)))

	failures := Has("a button", Object(right)).
		LeftOf(Object(left)).Check(In(root))
	require.Len(t, failures, 1)
	assert.Contains(t, failures[0].Message, "to the left of")
}

func TestBelowRelation(t *testing.T) {
	root, top, _, _, _ := tree()

	assert.Empty(t, Has("a button", Object(leftOf(root))).
		Below(Object(top)).Check(In(root)))

	failures := Has("a label", Object(top)).
		Below(Object(leftOf(root))).Check(In(root))
	require.Len(t, failures, 1)
	assert.Contains(t, failures[0].Message, "below")
}

func TestTopmostRelation(t *testing.T) {
	root, top, _, _, _ := tree()

	assert.Empty(t, Has("a label", Object(top)).Topmost().Check(In(root)))

	failures := Has("a button", Object(leftOf(root))).Topmost().Check(In(root))
	require.Len(t, failures, 1)
	assert.Contains(t, failures[0].Message, "topmost")
}

func TestBottommostRelation(t *testing.T) {
	root, _, _, _, middle := tree()

	assert.Empty(t, Has("a button", Object(middle)).Bottommost().Check(In(root)))

	failures := Has("a label", Object(leftOf(root))).Bottommost().Check(In(root))
	require.Len(t, failures, 1)
	assert.Contains(t, failures[0].Message, "bottommost")
}

func TestCenteredRelation(t *testing.T) {
	root, _, _, _, middle := tree()

	assert.Empty(t, Has("a button", Object(middle)).Centered().Check(In(root)))
}

func TestCenteredRelationFailsWhenOffCenter(t *testing.T) {
	root, _, left, _, _ := tree()

	failures := Has("a button", Object(left)).Centered().Check(In(root))
	require.Len(t, failures, 1)
	assert.Contains(t, failures[0].Message, "centered")
}

func TestRightOfCenterRelation(t *testing.T) {
	root, _, left, right, _ := tree()

	assert.Empty(t, Has("a button", Object(right)).RightOfCenter().Check(In(root)))
	assert.Empty(t, Has("a button", Object(left)).LeftOfCenter().Check(In(root)))

	failures := Has("a button", Object(right)).LeftOfCenter().Check(In(root))
	require.Len(t, failures, 1)
	assert.Contains(t, failures[0].Message, "left of the center")
}

func TestNotExists(t *testing.T) {
	root, _, _, _, _ := tree()

	assert.Empty(t, NotExists("a close button", Button("Close")).Check(In(root)))

	failures := NotExists("an open pack button", Button("Open pack")).Check(In(root))
	require.Len(t, failures, 1)
	assert.Contains(t, failures[0].Message, "found 1")
}

func TestObjectSelectorOnlyMatchesInsideTheScope(t *testing.T) {
	root, _, _, right, _ := tree()

	assert.Empty(t, Has("a button", Object(right)).Check(In(root)))

	failures := Has("a button", Object(right)).Check(In(container.NewWithoutLayout()))
	require.Len(t, failures, 1)
	assert.Contains(t, failures[0].Message, "found none")
}

func TestGroupPrefixesTheFailurePath(t *testing.T) {
	root, _, _, _, _ := tree()

	failures := Group("when the main window is shown",
		Has("a welcome header", Button("Welcome")),
	).Check(In(root))
	require.Len(t, failures, 1)
	assert.Equal(t, []string{"when the main window is shown", "a welcome header"}, failures[0].Path)
}

func TestRelationsDoNotMatchWhenTheReferenceIsMissing(t *testing.T) {
	root, _, left, _, _ := tree()

	failures := Has("a button", Object(left)).
		RightOf(Button("Close")).Check(In(root))
	require.Len(t, failures, 1)
	assert.Contains(t, failures[0].Message, "there is no")
}

func TestExactlySetsTheNumberOfMatches(t *testing.T) {
	root, _, _, _, _ := tree()

	openButtons := Any(Button("Open pack"), Button("Open folder"))

	assert.Empty(t, Has("an open button", openButtons).Exactly(2).Check(In(root)))

	failures := Has("an open button", openButtons).Exactly(1).Check(In(root))
	require.Len(t, failures, 1)
	assert.Contains(t, failures[0].Message, "exactly one widget")
}

func TestIconButtonSelector(t *testing.T) {
	icon := fyne.NewStaticResource("test.svg", []byte{})
	button := widget.NewButtonWithIcon("", icon, nil)
	root := container.NewWithoutLayout(button)
	root.Resize(fyne.NewSize(100, 100))

	assert.Empty(t, Has("an icon button", IconButton("test.svg")).Check(In(root)))

	failures := Has("an icon button", IconButton("other.svg")).Check(In(root))
	require.Len(t, failures, 1)
}

// leftOf returns the "Open pack" button of the tree.
func leftOf(root *fyne.Container) fyne.CanvasObject {
	return root.Objects[1]
}
