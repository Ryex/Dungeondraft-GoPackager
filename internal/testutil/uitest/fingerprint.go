package uitest

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
)

// LayoutFingerprint returns a description of the current layout of an object
// tree. It changes whenever a widget of the tree is moved or resized.
//
// Tests that wait for an interface to settle can use it to tell whether the
// layout is still changing.
func LayoutFingerprint(root fyne.CanvasObject) string {
	var fingerprint strings.Builder
	writeFingerprint(&fingerprint, root, "")
	return fingerprint.String()
}

func writeFingerprint(fingerprint *strings.Builder, o fyne.CanvasObject, indent string) {
	if o == nil {
		return
	}
	fmt.Fprintf(
		fingerprint, "%s%T pos=%v size=%v\n",
		indent, o, o.Position(), o.Size(),
	)
	for _, child := range children(o) {
		writeFingerprint(fingerprint, child, indent+"  ")
	}
}
