package uitest

import (
	"fmt"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
)

// Failure describes an expectation of a specification that was not met.
type Failure struct {
	// Path is the chain of the descriptions of the nodes leading to the
	// failure, outermost first.
	Path []string
	// Message describes what was expected and what was found.
	Message string
	// Details holds additional information, e.g. the widgets found in the
	// scope the expectation failed in.
	Details []string
}

// Expectation is a node of a specification tree. It looks for a widget and
// checks the constraints and the nested expectations that belong to it.
type Expectation struct {
	desc      string
	selector  Selector
	nested    []Node
	relations []Relation
	minimum   int
	maximum   int
}

// Has returns an expectation that exactly one widget of the scope matches the
// given selector. The nested expectations are checked against the matched
// widget.
func Has(desc string, sel Selector, nested ...Node) *Expectation {
	return &Expectation{
		desc:     desc,
		selector: sel,
		nested:   nested,
		minimum:  1,
		maximum:  1,
	}
}

// HasAny returns an expectation that at least one widget of the scope
// matches the given selector.
func HasAny(desc string, sel Selector, nested ...Node) *Expectation {
	return &Expectation{
		desc:     desc,
		selector: sel,
		nested:   nested,
		minimum:  1,
		maximum:  -1,
	}
}

// Exists returns an expectation that at least one widget of the scope
// matches the given selector.
func Exists(desc string, sel Selector) *Expectation {
	return HasAny(desc, sel)
}

// NotExists returns an expectation that no widget of the scope matches the
// given selector.
func NotExists(desc string, sel Selector) Node {
	return &absence{desc: desc, selector: sel}
}

// Exactly sets the number of widgets that are expected to match.
func (e *Expectation) Exactly(n int) *Expectation {
	c := e.copy()
	c.minimum = n
	c.maximum = n
	return c
}

// RightOf requires the widget to be placed to the right of a widget matching
// the given selector.
func (e *Expectation) RightOf(ref Selector) *Expectation {
	return e.withRelation(RightOf(ref))
}

// LeftOf requires the widget to be placed to the left of a widget matching
// the given selector.
func (e *Expectation) LeftOf(ref Selector) *Expectation {
	return e.withRelation(LeftOf(ref))
}

// Above requires the widget to be placed above a widget matching the given
// selector.
func (e *Expectation) Above(ref Selector) *Expectation {
	return e.withRelation(Above(ref))
}

// Below requires the widget to be placed below a widget matching the given
// selector.
func (e *Expectation) Below(ref Selector) *Expectation {
	return e.withRelation(Below(ref))
}

// Centered requires the widget to be horizontally centered in the scope.
func (e *Expectation) Centered() *Expectation {
	return e.withRelation(Centered())
}

// RightOfCenter requires the widget to be placed right of the vertical
// center line of the scope.
func (e *Expectation) RightOfCenter() *Expectation {
	return e.withRelation(RightOfCenter())
}

// LeftOfCenter requires the widget to be placed left of the vertical center
// line of the scope.
func (e *Expectation) LeftOfCenter() *Expectation {
	return e.withRelation(LeftOfCenter())
}

// Topmost requires the widget to be the topmost one of the scope.
func (e *Expectation) Topmost() *Expectation {
	return e.withRelation(Topmost())
}

// Bottommost requires the widget to be the bottommost one of the scope.
func (e *Expectation) Bottommost() *Expectation {
	return e.withRelation(Bottommost())
}

func (e *Expectation) withRelation(rel Relation) *Expectation {
	c := e.copy()
	c.relations = append(c.relations, rel)
	return c
}

func (e *Expectation) copy() *Expectation {
	c := *e
	c.relations = append([]Relation(nil), e.relations...)
	c.nested = append([]Node(nil), e.nested...)
	return &c
}

// String describes the expectation, for use in failure messages.
func (e *Expectation) String() string {
	return e.desc
}

// Check evaluates the expectation and returns the failures it found.
func (e *Expectation) Check(s *Scope) []Failure {
	candidates := e.selector.Select(s)

	var matched []fyne.CanvasObject
	var rejected []string
	for _, candidate := range candidates {
		if err := e.checkRelations(candidate, s); err != nil {
			rejected = append(rejected, fmt.Sprintf(
				"%s does not satisfy the constraint: %s",
				describeObject(candidate), err,
			))
			continue
		}
		matched = append(matched, candidate)
	}

	if len(matched) < e.minimum {
		return []Failure{{
			Path:    []string{e.desc},
			Message: e.notFoundMessage(len(candidates), rejected),
			Details: append(rejected, scopeDetails(s)...),
		}}
	}
	if e.maximum >= 0 && len(matched) > e.maximum {
		return []Failure{{
			Path: []string{e.desc},
			Message: fmt.Sprintf(
				"expected %s matching %s, found %d",
				e.countDescription(), e.describe(), len(matched),
			),
			Details: scopeDetails(s),
		}}
	}

	subject := matched[0]
	childScope := s
	if isContainer(subject) {
		childScope = &Scope{Root: subject, App: s.App}
	}

	var failures []Failure
	for _, nested := range e.nested {
		for _, failure := range nested.Check(childScope) {
			failure.Path = append([]string{e.desc}, failure.Path...)
			failures = append(failures, failure)
		}
	}
	return failures
}

func (e *Expectation) checkRelations(subject fyne.CanvasObject, s *Scope) error {
	for _, relation := range e.relations {
		if err := relation.Check(subject, s); err != nil {
			return err
		}
	}
	return nil
}

func (e *Expectation) countDescription() string {
	switch {
	case e.maximum < 0:
		return "at least one widget"
	case e.minimum == e.maximum:
		if e.minimum == 1 {
			return "exactly one widget"
		}
		return fmt.Sprintf("exactly %d widgets", e.minimum)
	default:
		return fmt.Sprintf("between %d and %d widgets", e.minimum, e.maximum)
	}
}

// describe renders the selector of the expectation together with the
// constraints it has to satisfy.
func (e *Expectation) describe() string {
	desc := e.selector.String()
	if len(e.relations) == 0 {
		return desc
	}
	parts := make([]string, 0, len(e.relations))
	for _, relation := range e.relations {
		parts = append(parts, relation.String())
	}
	return desc + " that is " + strings.Join(parts, " and ")
}

func (e *Expectation) notFoundMessage(candidateCount int, rejected []string) string {
	if candidateCount == 0 {
		return fmt.Sprintf(
			"expected %s matching %s, found none",
			e.countDescription(), e.describe(),
		)
	}
	message := fmt.Sprintf(
		"expected %s matching %s, but the %d candidate(s) found do not satisfy the constraints",
		e.countDescription(), e.describe(), candidateCount,
	)
	if len(rejected) > 0 {
		message += ": " + rejected[0]
	}
	return message
}

// absence is the expectation that nothing matches a selector.
type absence struct {
	desc     string
	selector Selector
}

func (a *absence) String() string {
	return a.desc
}

func (a *absence) Check(s *Scope) []Failure {
	found := a.selector.Select(s)
	if len(found) == 0 {
		return nil
	}
	details := make([]string, 0, len(found))
	for _, o := range found {
		details = append(details, "- "+describeObject(o))
	}
	return []Failure{{
		Path:    []string{a.desc},
		Message: fmt.Sprintf("expected no %s, found %d", a.selector.String(), len(found)),
		Details: details,
	}}
}

// group labels a set of expectations that share a condition.
type group struct {
	desc  string
	nodes []Node
}

// Group labels a set of expectations that share a condition. It does not
// change the scope of the expectations it holds.
func Group(desc string, nodes ...Node) Node {
	return &group{desc: desc, nodes: nodes}
}

func (g *group) String() string {
	return g.desc
}

func (g *group) Check(s *Scope) []Failure {
	var failures []Failure
	for _, node := range g.nodes {
		for _, failure := range node.Check(s) {
			failure.Path = append([]string{g.desc}, failure.Path...)
			failures = append(failures, failure)
		}
	}
	return failures
}

// Expect evaluates the given expectations against the scope and reports
// every expectation that is not met.
func Expect(t *testing.T, scope *Scope, nodes ...Node) {
	t.Helper()

	var failures []Failure
	for _, node := range nodes {
		failures = append(failures, node.Check(scope)...)
	}
	if len(failures) == 0 {
		return
	}

	var report strings.Builder
	report.WriteString("the UI specification is not satisfied:\n")
	for _, failure := range failures {
		report.WriteString(formatFailure(failure))
	}
	t.Error(report.String())
}

func formatFailure(failure Failure) string {
	var report strings.Builder
	indent := ""
	for _, step := range failure.Path {
		report.WriteString(indent + "- " + step + "\n")
		indent += "  "
	}
	report.WriteString(indent + failure.Message + "\n")
	for _, detail := range failure.Details {
		report.WriteString(indent + "  " + detail + "\n")
	}
	return report.String()
}

// maxScopeDetails is the number of widgets listed when an expectation fails.
const maxScopeDetails = 30

func scopeDetails(s *Scope) []string {
	var details []string
	for _, o := range objects(s.Root) {
		if !o.Visible() {
			continue
		}
		details = append(details, "- "+describeObject(o))
		if len(details) >= maxScopeDetails {
			details = append(details, "- ...")
			break
		}
	}
	return details
}
