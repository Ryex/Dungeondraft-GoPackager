package gui

import (
	"testing"

	"github.com/ryex/dungeondraft-gopackager/internal/testutil/uitest"
)

// TestSpecPackFolder verifies the "when a pack folder is opened" section of
// the UI specification.
//
// Covered by this test:
//
//	when a pack folder is opened
//	  - the main content shows activity content until it loads
//	  - when the pack is loaded
//	    - the main content shows pack content
//	      - with a button panel on top
//	        - with a output path entry
//	        - with a browse button
//	        - with a thumbnail generate button
//	        - with a edit pack info button
//	        - with a generate tags button
//	        - with a edit tag sets button
//	        - with an overwrite check box
//	        - with a package button
//	      - with a split pane below the button pane
//	        - with a package resource tree on the left
//	        - with a resource pane on the right
func TestSpecPackFolder(t *testing.T) {
	a := newSpecApp(t)
	texts := specTexts()
	dir := packFixtureDir(t)

	// the load is blocked, so that the state of the interface while the
	// pack is loading can be observed
	release := blockUnpackedLoad(a)
	a.loadUnpackedPath(dir)

	uitest.Expect(t, uitest.In(namedPart(t, a, nameContent)),
		uitest.Group("when a pack folder is opened",
			uitest.Has("activity content", uitest.Activity()),
			uitest.Has("a progress bar", uitest.ProgressBar()),
		),
	)

	release()
	waitForPackLoad(t, a)

	uitest.Expect(t, uitest.In(namedPart(t, a, nameContent)),
		uitest.Group("when the pack is loaded",
			uitest.Has("a button panel on top", uitest.Named(namePackPanel)).
				Topmost(),
			uitest.Has("a split pane below the button panel",
				uitest.Named(nameSplitPane)).
				Below(uitest.Named(namePackPanel)),
			uitest.Has("a package resource tree on the left",
				uitest.Named(nameResourceTree)).
				LeftOf(uitest.Named(nameResourcePane)),
			uitest.Has("a resource pane on the right",
				uitest.Named(nameResourcePane)).
				RightOf(uitest.Named(nameResourceTree)),
		),
	)

	uitest.Expect(t, uitest.In(namedPart(t, a, namePackPanel)),
		uitest.Group("when a pack folder is opened",
			uitest.Has("an output path entry",
				uitest.Entry(texts["outPath"]),
			),
			uitest.Has("a browse button to the right of the output path entry",
				uitest.Button(texts["browse"])).
				RightOf(uitest.Entry(texts["outPath"])),
			uitest.Has("a thumbnail generate button",
				uitest.Button(texts["thumbnails"]),
			),
			uitest.Has("an edit pack info button",
				uitest.Button(texts["editPack"]),
			),
			uitest.Has("a generate tags button",
				uitest.Button(texts["generateTags"]),
			),
			uitest.Has("an edit tag sets button",
				uitest.Button(texts["tagSets"]),
			),
			uitest.Has("an overwrite check box",
				uitest.Check(texts["overwrite"]),
			),
			uitest.Has("a package button",
				uitest.Button(texts["package"]),
			),
		),
	)
}
