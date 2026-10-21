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

	uitest.Expect(t, uitest.In(namedPart(t, a, nameMainColumn)),
		uitest.Group("when the main window is shown",
			uitest.Has("a welcome header at the top", uitest.Named(nameHeader)).
				Topmost(),
			uitest.Has("a path input section below the welcome header",
				uitest.Named(namePathSection)).
				Below(uitest.Named(nameHeader)),
			uitest.Has("a content section on the bottom", uitest.Named(nameContent)).
				Bottommost(),
		),
	)

	uitest.Expect(t, uitest.In(namedPart(t, a, nameHeader)),
		uitest.Group("when the main window is shown",
			uitest.Has("a theme toggle on the left", uitest.Named(nameThemeToggle)).
				LeftOfCenter(),
			uitest.Has("a language select dropdown to the right of the theme toggle",
				uitest.Named(nameLanguageSelect)).
				RightOf(uitest.Named(nameThemeToggle)),
			uitest.Has("a title in the center", uitest.Named(nameTitleText)).
				Centered(),
			uitest.Has("a subtitle in the center", uitest.Named(nameSubTitleText)).
				Centered(),
			uitest.Has("an about icon button to the right", uitest.Named(nameAboutButton)).
				RightOfCenter(),
			uitest.Has("a github icon button to the right of the about button",
				uitest.Named(nameGithubButton)).
				RightOf(uitest.Named(nameAboutButton)),
			uitest.Has("an info icon button to the right of the github button",
				uitest.Named(nameInfoButton)).
				RightOf(uitest.Named(nameGithubButton)),
		),
	)

	uitest.Expect(t, uitest.In(namedPart(t, a, namePathSection)),
		uitest.Group("when the main window is shown",
			uitest.Has("a path input entry to the left", uitest.Named(namePathEntry)).
				LeftOf(uitest.Named(nameOpenPackButton)),
			uitest.Has("an Open Pack button to the right of the path entry",
				uitest.Named(nameOpenPackButton)).
				RightOf(uitest.Named(namePathEntry)),
			uitest.Has("an Open Folder button to the right of the Open Pack button",
				uitest.Named(nameOpenFolderButton)).
				RightOf(uitest.Named(nameOpenPackButton)),
		),
	)

	// the widgets of the header and of the path section have to be the ones
	// the specification names
	uitest.Expect(t, uitest.In(namedPart(t, a, namePathSection)),
		uitest.Has("the path input entry", uitest.Entry(texts["pathPlaceholder"])),
		uitest.Has("the Open Pack button", uitest.Button(texts["openPack"])),
		uitest.Has("the Open Folder button", uitest.Button(texts["openFolder"])),
	)
}
