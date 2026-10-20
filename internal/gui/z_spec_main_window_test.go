package gui

import (
	"testing"

	"github.com/ryex/dungeondraft-gopackager/internal/testutil/uitest"
)

// TestSpecMainWindow verifies the "when the main window is shown" section of
// the UI specification.
//
// Covered by this test:
//
//	when the main window is shown
//	  - there is a welcome header at the top
//	    - with a theme toggle on the left
//	      - with a language select dropdown to the right of the theme toggle
//	    - with a title and subtitle in the center
//	    - with a "about" ion button to the right
//	      - with a github icon button to the right of "about"
//	        - with a "info" icon button to the right of "github"
//	  - with a path input section below the welcome header
//	    - with a path input entry to the left
//	    - with a "Open Pack" button to the right of the path entry
//	    - with a "Open Folder" button to the right of the "Open Pack" button
//	  - with a content section on the bottom
func TestSpecMainWindow(t *testing.T) {
	a := newSpecApp(t)
	texts := specTexts()

	uitest.Expect(t, uitest.In(a.ui.MainColumn),
		uitest.Group("when the main window is shown",
			uitest.Has("a welcome header at the top", uitest.Object(a.ui.Header)).
				Topmost(),
			uitest.Has("a path input section below the welcome header",
				uitest.Object(a.ui.PathSection)).
				Below(uitest.Object(a.ui.Header)),
			uitest.Has("a content section on the bottom", uitest.Object(a.ui.Content)).
				Bottommost(),
		),
	)

	uitest.Expect(t, uitest.In(a.ui.Header),
		uitest.Group("when the main window is shown",
			uitest.Has("a theme toggle on the left", uitest.Object(a.ui.ThemeToggle)).
				LeftOfCenter(),
			uitest.Has("a language select dropdown to the right of the theme toggle",
				uitest.Object(a.ui.LanguageSelect)).
				RightOf(uitest.Object(a.ui.ThemeToggle)),
			uitest.Has("a title in the center", uitest.Object(a.ui.TitleText)).
				Centered(),
			uitest.Has("a subtitle in the center", uitest.Object(a.ui.SubTitleText)).
				Centered(),
			uitest.Has("an about icon button to the right", uitest.Object(a.ui.AboutButton)).
				RightOfCenter(),
			uitest.Has("a github icon button to the right of the about button",
				uitest.Object(a.ui.GithubButton)).
				RightOf(uitest.Object(a.ui.AboutButton)),
			uitest.Has("an info icon button to the right of the github button",
				uitest.Object(a.ui.InfoButton)).
				RightOf(uitest.Object(a.ui.GithubButton)),
		),
	)

	uitest.Expect(t, uitest.In(a.ui.PathSection),
		uitest.Group("when the main window is shown",
			uitest.Has("a path input entry to the left", uitest.Object(a.ui.PathEntry)).
				LeftOf(uitest.Object(a.ui.OpenPackButton)),
			uitest.Has("an Open Pack button to the right of the path entry",
				uitest.Object(a.ui.OpenPackButton)).
				RightOf(uitest.Object(a.ui.PathEntry)),
			uitest.Has("an Open Folder button to the right of the Open Pack button",
				uitest.Object(a.ui.OpenFolderButton)).
				RightOf(uitest.Object(a.ui.OpenPackButton)),
		),
	)

	// the widgets of the header and of the path section have to be the ones
	// the specification names
	uitest.Expect(t, uitest.In(a.ui.PathSection),
		uitest.Has("the path input entry", uitest.Entry(texts["pathPlaceholder"])),
		uitest.Has("the Open Pack button", uitest.Button(texts["openPack"])),
		uitest.Has("the Open Folder button", uitest.Button(texts["openFolder"])),
	)
}
