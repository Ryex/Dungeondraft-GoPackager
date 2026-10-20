package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/ryex/dungeondraft-gopackager/internal/gui/resource_tree"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/widgets"
)

// uiRegions holds named references to the widgets and the regions that make
// up the user interface.
//
// The user interface specification names the parts of the interface, so the
// code names them too. This keeps the layout introspectable, which the UI
// specification tests rely on.
type uiRegions struct {
	// MainColumn is the vertical column of the main window. It holds the
	// welcome header, the path input section and the main content.
	MainColumn *fyne.Container

	// Header is the welcome header of the main window.
	Header *fyne.Container
	// ThemeToggle switches between the light and the dark theme.
	ThemeToggle *widgets.Toggle
	// LanguageSelect is the drop down that selects the language.
	LanguageSelect *widget.Select
	// TitleText is the title shown in the welcome header.
	TitleText *widgets.ThemedText
	// SubTitleText is the subtitle shown in the welcome header.
	SubTitleText *widgets.ThemedText
	// AboutButton opens the about dialog.
	AboutButton *widget.Button
	// GithubButton opens the project page.
	GithubButton *widget.Button
	// InfoButton opens the credits window.
	InfoButton *widget.Button

	// PathSection is the section holding the path input.
	PathSection *fyne.Container
	// PathEntry is the entry the path of the pack is typed into.
	PathEntry *widget.Entry
	// OpenPackButton opens a packed resource pack file.
	OpenPackButton *widget.Button
	// OpenFolderButton opens an unpacked resource pack folder.
	OpenFolderButton *widget.Button

	// Content holds the main content of the window.
	Content *fyne.Container

	// ButtonPanel is the panel of buttons shown above the resource tree
	// while a package is loaded.
	ButtonPanel *fyne.Container
	// SplitPane splits the resource tree from the resource pane.
	SplitPane *fyne.Container
	// ResourceTree is the tree of the resources of the loaded package.
	ResourceTree *resource_tree.ResourceTree
	// ResourcePane shows the details of the selected resource.
	ResourcePane *fyne.Container
}

// regions returns the named references to the current layout, creating them
// when the layout has not been built yet.
func (a *App) regions() *uiRegions {
	if a.ui == nil {
		a.ui = &uiRegions{}
	}
	return a.ui
}
