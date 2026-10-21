package uitest

import (
	"fmt"
	"math"

	"fyne.io/fyne/v2"
)

// epsilon is the tolerance used when comparing pixel positions.
const epsilon = 0.01

// centerTolerance is the tolerance used when comparing the horizontal
// center of a widget with the horizontal center of its scope.
const centerTolerance = 1.0

type side int

const (
	sideRight side = iota
	sideLeft
	sideAbove
	sideBelow
)

func (s side) String() string {
	switch s {
	case sideRight:
		return "to the right of"
	case sideLeft:
		return "to the left of"
	case sideAbove:
		return "above"
	case sideBelow:
		return "below"
	}
	return "next to"
}

// sideRelation asserts that the subject is placed on a given side of another
// widget.
type sideRelation struct {
	ref  Selector
	side side
}

// RightOf asserts that the subject is placed to the right of a widget
// matching the given selector.
func RightOf(ref Selector) Relation {
	return sideRelation{ref: ref, side: sideRight}
}

// LeftOf asserts that the subject is placed to the left of a widget
// matching the given selector.
func LeftOf(ref Selector) Relation {
	return sideRelation{ref: ref, side: sideLeft}
}

// Above asserts that the subject is placed above a widget matching the given
// selector.
func Above(ref Selector) Relation {
	return sideRelation{ref: ref, side: sideAbove}
}

// Below asserts that the subject is placed below a widget matching the given
// selector.
func Below(ref Selector) Relation {
	return sideRelation{ref: ref, side: sideBelow}
}

func (r sideRelation) Check(subject fyne.CanvasObject, s *Scope) error {
	refs := r.ref.Select(s)
	if len(refs) == 0 {
		return fmt.Errorf("there is no %s to compare with", r.ref.String())
	}

	subjectPos, ok := s.Position(subject)
	if !ok {
		return fmt.Errorf("the subject is not part of the scope")
	}
	subjectSize := subject.Size()

	for _, ref := range refs {
		refPos, ok := s.Position(ref)
		if !ok {
			continue
		}
		refSize := ref.Size()

		var satisfied bool
		switch r.side {
		case sideRight:
			satisfied = subjectPos.X >= refPos.X+refSize.Width-epsilon
		case sideLeft:
			satisfied = subjectPos.X+subjectSize.Width <= refPos.X+epsilon
		case sideAbove:
			satisfied = subjectPos.Y+subjectSize.Height <= refPos.Y+epsilon
		case sideBelow:
			satisfied = subjectPos.Y >= refPos.Y+refSize.Height-epsilon
		}
		if satisfied {
			return nil
		}
	}

	return fmt.Errorf("expected to be %s %s", r.side, r.ref.String())
}

func (r sideRelation) String() string {
	return fmt.Sprintf("%s %s", r.side, r.ref.String())
}

// centeredRelation asserts that the subject is horizontally centered in its
// scope.
type centeredRelation struct{}

// Centered asserts that the subject is horizontally centered in the scope.
func Centered() Relation {
	return centeredRelation{}
}

func (r centeredRelation) Check(subject fyne.CanvasObject, s *Scope) error {
	subjectCenter, scopeCenter, err := centers(subject, s)
	if err != nil {
		return err
	}
	if math.Abs(float64(subjectCenter-scopeCenter)) > centerTolerance {
		return fmt.Errorf(
			"expected to be centered in the scope, but its center is at %.1f instead of %.1f",
			subjectCenter, scopeCenter,
		)
	}
	return nil
}

func (r centeredRelation) String() string {
	return "centered in the scope"
}

// halfRelation asserts that the subject is left or right of the vertical
// center line of its scope.
type halfRelation struct {
	right bool
}

// RightOfCenter asserts that the subject is placed right of the vertical
// center line of the scope.
func RightOfCenter() Relation {
	return halfRelation{right: true}
}

// LeftOfCenter asserts that the subject is placed left of the vertical
// center line of the scope.
func LeftOfCenter() Relation {
	return halfRelation{right: false}
}

func (r halfRelation) Check(subject fyne.CanvasObject, s *Scope) error {
	subjectCenter, scopeCenter, err := centers(subject, s)
	if err != nil {
		return err
	}
	if r.right && subjectCenter <= scopeCenter {
		return fmt.Errorf(
			"expected to be right of the center of the scope, but its center is at %.1f instead of %.1f",
			subjectCenter, scopeCenter,
		)
	}
	if !r.right && subjectCenter >= scopeCenter {
		return fmt.Errorf(
			"expected to be left of the center of the scope, but its center is at %.1f instead of %.1f",
			subjectCenter, scopeCenter,
		)
	}
	return nil
}

func (r halfRelation) String() string {
	if r.right {
		return "right of the center of the scope"
	}
	return "left of the center of the scope"
}

func centers(subject fyne.CanvasObject, s *Scope) (float32, float32, error) {
	subjectPos, ok := s.Position(subject)
	if !ok {
		return 0, 0, fmt.Errorf("the subject is not part of the scope")
	}
	subjectCenter := subjectPos.X + subject.Size().Width/2
	scopeCenter := s.Root.Size().Width / 2
	return subjectCenter, scopeCenter, nil
}

// visibilityRelation asserts that a widget is shown or hidden.
type visibilityRelation struct {
	visible bool
}

// Visible asserts that the widget is shown.
func Visible() Relation {
	return visibilityRelation{visible: true}
}

// Hidden asserts that the widget is hidden.
func Hidden() Relation {
	return visibilityRelation{visible: false}
}

func (r visibilityRelation) Check(subject fyne.CanvasObject, s *Scope) error {
	if subject.Visible() == r.visible {
		return nil
	}
	if r.visible {
		return fmt.Errorf("expected to be visible, but it is hidden")
	}
	return fmt.Errorf("expected to be hidden, but it is visible")
}

func (r visibilityRelation) String() string {
	if r.visible {
		return "visible"
	}
	return "hidden"
}

// extremeRelation asserts that no other widget of the scope is placed
// further up or further down.
type extremeRelation struct {
	top bool
}

// Topmost asserts that no widget of the scope is placed above the subject.
func Topmost() Relation {
	return extremeRelation{top: true}
}

// Bottommost asserts that no widget of the scope is placed below the
// subject.
func Bottommost() Relation {
	return extremeRelation{top: false}
}

func (r extremeRelation) Check(subject fyne.CanvasObject, s *Scope) error {
	subjectPos, ok := s.Position(subject)
	if !ok {
		return fmt.Errorf("the subject is not part of the scope")
	}
	subjectSize := subject.Size()

	// the ancestors of the subject hold it, so they can never be above or
	// below it
	enclosing := map[fyne.CanvasObject]bool{}
	for _, ancestor := range ancestorsOf(s.Root, subject) {
		enclosing[ancestor] = true
	}

	for _, o := range objects(s.Root) {
		if o == subject || enclosing[o] || !o.Visible() {
			continue
		}
		pos, ok := s.Position(o)
		if !ok {
			continue
		}
		if r.top {
			if pos.Y < subjectPos.Y-epsilon {
				return fmt.Errorf(
					"expected to be the topmost widget of the scope, but %s is above it",
					describeObject(o),
				)
			}
			continue
		}
		if pos.Y+o.Size().Height > subjectPos.Y+subjectSize.Height+epsilon {
			return fmt.Errorf(
				"expected to be the bottommost widget of the scope, but %s is below it",
				describeObject(o),
			)
		}
	}
	return nil
}

func (r extremeRelation) String() string {
	if r.top {
		return "at the top of the scope"
	}
	return "at the bottom of the scope"
}
