package gui

import (
	"testing"

	"github.com/ryex/dungeondraft-gopackager/internal/testutil/uitest"
)

// TestSpecDefaultContent verifies the "when the app has just started and no
// pack has been opened yet" section of the UI specification.
//
// Covered by this test:
//
//	when the app has just started and no pack has been opened yet
//	  - the main content shows main.defaultText "Enter a path to get started." ...
func TestSpecDefaultContent(t *testing.T) {
	a := newSpecApp(t)

	uitest.Expect(t, uitest.In(a.ui.Content),
		uitest.Group("when the app has just started and no pack has been opened yet",
			uitest.Has("the main default text",
				uitest.ThemedText(defaultTextLines()[0]),
			),
		),
	)

	// the whole of main.defaultText is shown, split over its lines
	for _, line := range defaultTextLines() {
		uitest.Expect(t, uitest.In(a.ui.Content),
			uitest.Has("the line of the main default text",
				uitest.ThemedText(line),
			),
		)
	}
}
