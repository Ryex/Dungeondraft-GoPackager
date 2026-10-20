package gui

import (
	"testing"

	"github.com/ryex/dungeondraft-gopackager/internal/testutil/uitest"
)

// TestSpecPackFile verifies the "when a packed pack file is opened" section
// of the UI specification.
//
// Covered by this test:
//
//	when a packed pack file is opened
//	  - the main content shows activity content until it loads
//	  - when the pack is loaded
//	    - the main content shows pack content
//	      - with a button panel on top
//	        - with a output path entry
//	        - with a browse button
//	        - with a package information button
//	        - with a view tag sets button
//	        - with an overwrite check box
//	        - with a rip textures check box
//	        - with an extract thumbnails check box
//	        - with an extract button
//	      - with a split pane below the button pane
//	        - with a package resource tree on the left
//	        - with a resource pane on the right
func TestSpecPackFile(t *testing.T) {
	// the fixture is created before the app, so that the package is closed
	// before the temporary directory of the fixture is removed
	packFile := packedFixtureFile(t)

	a := newSpecApp(t)
	texts := specTexts()

	// the load is blocked, so that the state of the interface while the
	// pack is loading can be observed
	release := blockPackedLoad(a)
	a.loadPack(packFile)

	uitest.Expect(t, uitest.In(a.ui.Content),
		uitest.Group("when a packed pack file is opened",
			uitest.Has("activity content", uitest.Activity()),
			uitest.Has("a progress bar", uitest.ProgressBar()),
		),
	)

	release()
	waitForPackLoad(t, a)

	uitest.Expect(t, uitest.In(a.ui.Content),
		uitest.Group("when the pack is loaded",
			uitest.Has("a button panel on top", uitest.Object(a.ui.ButtonPanel)).
				Topmost(),
			uitest.Has("a split pane below the button panel",
				uitest.Object(a.ui.SplitPane)).
				Below(uitest.Object(a.ui.ButtonPanel)),
			uitest.Has("a package resource tree on the left",
				uitest.Object(a.ui.ResourceTree)).
				LeftOf(uitest.Object(a.ui.ResourcePane)),
			uitest.Has("a resource pane on the right",
				uitest.Object(a.ui.ResourcePane)).
				RightOf(uitest.Object(a.ui.ResourceTree)),
		),
	)

	uitest.Expect(t, uitest.In(a.ui.ButtonPanel),
		uitest.Group("when a packed pack file is opened",
			uitest.Has("an output path entry",
				uitest.Entry(texts["unpackOutPath"]),
			),
			uitest.Has("a browse button to the right of the output path entry",
				uitest.Button(texts["browse"])).
				RightOf(uitest.Entry(texts["unpackOutPath"])),
			uitest.Has("a package information button",
				uitest.Button(texts["packageInfo"]),
			),
			uitest.Has("a view tag sets button",
				uitest.Button(texts["viewTagSets"]),
			),
			uitest.Has("an overwrite check box",
				uitest.Check(texts["unpackOverwrite"]),
			),
			uitest.Has("a rip textures check box",
				uitest.Check(texts["ripTextures"]),
			),
			uitest.Has("an extract thumbnails check box",
				uitest.Check(texts["extractThumbnails"]),
			),
			uitest.Has("an extract button",
				uitest.Button(texts["extract"]),
			),
		),
	)
}
