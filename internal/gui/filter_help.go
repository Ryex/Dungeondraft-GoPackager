package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ryex/dungeondraft-gopackager/internal/gui/lang"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/named"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/widgets"
)

// The names of the parts of the help dialog of the filter language.
//
// The parts of the interface are named where they are created, so that they
// can be looked up later, e.g. by the specification tests.
const (
	nameFilterHelpSectionList = "filterHelpSectionList"
	nameFilterHelpScroll      = "filterHelpScroll"
)

// filterHelpSection describes one kind of filter of the filter language.
//
// The description of a section is written in markdown and translated, so it
// has to be short enough to be translated reasonably. The examples hold the
// syntax of the filters only, so they are not translated.
type filterHelpSection struct {
	// titleKey is the translation key of the title of the section.
	titleKey string
	// title is the fallback of the title of the section.
	title string
	// textKey is the translation key of the markdown description of the
	// section.
	textKey string
	// text is the fallback of the markdown description of the section.
	text string
	// examples are the examples of the section, as markdown. They hold the
	// syntax of the filters only, so they are not translated.
	examples string
}

// filterHelpSections returns the sections of the help of the filter language,
// in the order they are shown in the help dialog.
func filterHelpSections() []filterHelpSection {
	return []filterHelpSection{
		{
			titleKey: "tree.filter.help.name.title",
			title:    "Name",
			textKey:  "tree.filter.help.name.text",
			text: "Matches the name of a resource, without its extension. " +
				"The name is matched anywhere in it, unless it is prefixed " +
				"with `^` to match the beginning, `$` to match the end or " +
				"`=` to match the whole name. A `%` prefix makes the match " +
				"case sensitive.",
			examples: "rock\n%Rock\n^Rock\n$Rock\n=Rock",
		},
		{
			titleKey: "tree.filter.help.tag.title",
			title:    "Tag",
			textKey:  "tree.filter.help.tag.text",
			text: "Matches a tag of a resource. The tag is matched anywhere " +
				"in it, unless it is prefixed with `^` to match the beginning, " +
				"`$` to match the end or `=` to match the whole tag. A `%` " +
				"prefix makes the match case sensitive.",
			examples: "#rock\n#%Rock\n#^Rock\n#$Rock\n#=Rock",
		},
		{
			titleKey: "tree.filter.help.path.title",
			title:    "Path",
			textKey:  "tree.filter.help.path.text",
			text: "Matches the path of a resource, relative to the pack, " +
				"with a glob pattern. `*` matches any part of a name and " +
				"`**` matches any number of folders.",
			examples: "/textures/objects/**\n/textures/*/*.png",
		},
		{
			titleKey: "tree.filter.help.kind.title",
			title:    "Kind",
			textKey:  "tree.filter.help.kind.text",
			text: "Matches the kind of a resource. The available kinds are " +
				"listed below.",
			examples: "@IsTexture\n@IsObject\n@IsWall\n@IsTileset\n" +
				"@IsCave\n@IsLight\n@IsMaterial\n@IsPath\n@IsPattern\n" +
				"@IsPortal\n@IsTerrain\n@IsThumbnail\n@IsData\n@IsMetadata\n" +
				"@IsWallData\n@IsTilesetData\n@IsTaggable",
		},
		{
			titleKey: "tree.filter.help.combine.title",
			title:    "Combining",
			textKey:  "tree.filter.help.combine.text",
			text: "Several filters are combined with OR by default, so a " +
				"resource matches when it matches any of them. A `&` " +
				"combines the filters with AND instead. Filters can be " +
				"grouped with parentheses and a `!` negates a filter.",
			examples: "rock tree\n& rock tree\n!rock\n(rock tree)",
		},
		{
			titleKey: "tree.filter.help.quoting.title",
			title:    "Quoting",
			textKey:  "tree.filter.help.quoting.text",
			text: "A name or a tag that contains spaces or special " +
				"characters can be put in quotes. A `\\` escapes the " +
				"character that follows it.",
			examples: "\"two words\"\ntwo\\ words",
		},
	}
}

// filterHelpButton returns the button that opens the help of the filter
// language.
func (a *App) filterHelpButton() fyne.CanvasObject {
	btn := widgets.NewToolTipButtonWithIcon("", theme.QuestionIcon(), func() {
		a.showFilterHelp()
	})
	btn.SetToolTip(lang.X("tree.filter.help.tooltip", "Show the filter language"))
	return btn
}

// showFilterHelp shows the dialog that describes the filter language.
//
// The help is longer than the dialog, so it is scrollable. The list of the
// sections on the left scrolls the help to the section that is clicked.
func (a *App) showFilterHelp() {
	sections := filterHelpSections()

	// the titles of the sections, so that the list of the sections can
	// scroll the help to the section that is clicked
	anchors := make([]fyne.CanvasObject, 0, len(sections))
	content := container.NewVBox()

	for _, section := range sections {
		title := widget.NewRichTextFromMarkdown("## " + lang.X(section.titleKey, section.title))
		title.Wrapping = fyne.TextWrapWord

		text := widget.NewRichTextFromMarkdown(lang.X(section.textKey, section.text))
		text.Wrapping = fyne.TextWrapWord

		// the examples hold the syntax of the filters only, so they are
		// rendered as a code block and are not translated
		examples := widget.NewRichTextFromMarkdown("```\n" + section.examples + "\n```")
		examples.Wrapping = fyne.TextWrapWord

		content.Add(title)
		content.Add(text)
		content.Add(examples)

		anchors = append(anchors, title)
	}

	scroll := container.NewVScroll(container.NewPadded(content))

	// the list of the sections, which scrolls the help to the section that
	// is clicked
	sectionList := widget.NewList(
		func() int {
			return len(sections)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(lang.X(sections[id].titleKey, sections[id].title))
		},
	)
	sectionList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(anchors) {
			return
		}
		// the section is scrolled to the top of the help, so that its title
		// is visible. The position of the title is relative to the box of the
		// sections, which is padded inside the scrollable content.
		scroll.ScrollToOffset(fyne.NewPos(
			0,
			anchors[id].Position().Y+content.Position().Y,
		))
	}

	nav := container.NewHSplit(
		named.New(nameFilterHelpSectionList, sectionList),
		named.New(nameFilterHelpScroll, scroll),
	)
	nav.Offset = 0.25

	dlg := dialog.NewCustom(
		lang.X("tree.filter.help.title", "Filter Language"),
		lang.X("tree.filter.help.close", "Close"),
		container.NewPadded(nav),
		a.window,
	)
	dlg.Resize(fyne.NewSize(760, 620))
	dlg.Show()
}
