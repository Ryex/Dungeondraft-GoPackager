package gui

import (
	"maps"
	"os"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	xlayout "fyne.io/x/fyne/layout"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/bindings"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/layouts"
	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
	log "github.com/sirupsen/logrus"
)

func (a *App) createTagGenDialog() dialog.Dialog {
	examplePathParts := []string{"textures", "objects", "{Set A} Tag A", "{Set B} {Set C} Long Tag B", "Tag C", "object.png"}

	var (
		tags    []string
		tagSets []string
	)

	tagsMap := make(map[string]*structures.Set[string])

	selectedTag := ""
	boundSelectedTag := binding.BindString(&selectedTag)

	boundTags := binding.BindStringList(&tags)
	boundTagSets := binding.BindStringList(&tagSets)

	tagsList := widget.NewListWithData(
		boundTags,
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			l := co.(*widget.Label)
			l.Bind(di.(binding.String))
		},
	)

	selectedTagID := -1

	tagsList.OnSelected = func(id widget.ListItemID) {
		selectedTagID = id
		boundSelectedTag.Set(tags[id])
	}

	setsList := widget.NewListWithData(
		boundTagSets,
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			l := co.(*widget.Label)
			l.Bind(di.(binding.String))
		},
	)

	updateSets := func() {
		ts, ok := tagsMap[selectedTag]
		if ok && ts != nil {
			tagSets = ts.AsSlice()
		} else {
			tagSets = []string{}
		}
		slices.Sort(tagSets)
		boundTagSets.Reload()
	}

	boundSelectedTag.AddListener(binding.NewDataListener(updateSets))

	updateTags := func() {
		tags = utils.MapKeys(tagsMap)
		slices.Sort(tags)
		boundTags.Reload()
		if len(tags) > 0 {
			if selectedTagID >= 0 {
				boundSelectedTag.Set(tags[selectedTagID])
				tagsList.Select(selectedTagID)
			} else {
				selectedTagID = 0
				boundSelectedTag.Set(tags[0])
				tagsList.Select(0)
			}
		} else {
			selectedTagID = -1
			boundSelectedTag.Set("")
		}
	}

	var (
		buildGlobalTagSet     = false
		globalTagSet          = a.pkg.Name()
		buildTagSetFrpmPrefix = true
		prefixSplitMode       = false
		prefixSplitSeparator  = "|"
		tagSetPrefixDelimiter = [2]string{"{", "}"}
		stripTagSetPrefix     = true
		stripExtraPrefix      = ""
	)
	generateOptions := &ddpackage.GenerateTagsOptions{
		BuildGlobalTagSet:      buildGlobalTagSet,
		GlobalTagSet:           globalTagSet,
		BuildTagSetsFromPrefix: buildTagSetFrpmPrefix,
		PrefixSplitMode:        prefixSplitMode,
		TagSetPrefrixDelimiter: tagSetPrefixDelimiter,
		StripTagSetPrefix:      stripTagSetPrefix,
		StripExtraPrefix:       stripExtraPrefix,
	}
	generator := ddpackage.NewGenerateTags(generateOptions)

	boundBuildGlobalTagSet := binding.BindBool(&buildGlobalTagSet)
	boundGlobalTagSet := binding.BindString(&globalTagSet)

	boundBuildTagSetsFromPrefix := binding.BindBool(&buildTagSetFrpmPrefix)

	boundPrefixSplitMode := binding.BindBool(&prefixSplitMode)
	boundPrefixSplitSeparator := binding.BindString(&prefixSplitSeparator)
	boundPFDStart := binding.BindString(&tagSetPrefixDelimiter[0])
	boundPFDStop := binding.BindString(&tagSetPrefixDelimiter[1])

	boundStripTagSetPrefix := binding.BindBool(&stripTagSetPrefix)

	boundStripExtraPrefix := binding.BindString(&stripExtraPrefix)

	lastSplitSeperator := prefixSplitSeparator
	lastDelimiter := [2]string{tagSetPrefixDelimiter[0], tagSetPrefixDelimiter[1]}

	updateTagsMap := func() {
		generateOptions = &ddpackage.GenerateTagsOptions{
			BuildGlobalTagSet:      buildGlobalTagSet,
			GlobalTagSet:           globalTagSet,
			BuildTagSetsFromPrefix: buildTagSetFrpmPrefix,
			PrefixSplitMode:        prefixSplitMode,
			TagSetPrefrixDelimiter: func() [2]string {
				if prefixSplitMode {
					return [2]string{prefixSplitSeparator, ""}
				}
				return tagSetPrefixDelimiter
			}(),
			StripTagSetPrefix: stripTagSetPrefix,
			StripExtraPrefix:  stripExtraPrefix,
		}
		generator = ddpackage.NewGenerateTags(generateOptions)
		tagsMap = generator.TagsFromPath(strings.Join(examplePathParts, "/"))
		updateTags()
		updateSets()
	}
	updateTagsMap()

	bindings.AddListenerToAll(
		updateTagsMap,
		boundBuildGlobalTagSet,
		boundGlobalTagSet,
		boundBuildTagSetsFromPrefix,
		boundPrefixSplitMode,
		boundPrefixSplitSeparator,
		boundPFDStart,
		boundPFDStop,
		boundStripTagSetPrefix,
		boundStripExtraPrefix,
	)

	examplePathLbl := widget.NewLabel(
		lang.X("pathGen.examplePath.label", "Example Path"),
	)
	examplePathEntry := widget.NewEntry()
	examplePathEntry.SetText(strings.Join(examplePathParts, string(os.PathSeparator)))

	examplePathEntry.OnChanged = func(path string) {
		path = strings.ReplaceAll(path, string(os.PathSeparator), "/")
		parts := strings.Split(path, "/")

		if len(parts) < 3 {
			examplePathParts = []string{"textures", "objects", "object.png"}
			examplePathEntry.SetText(strings.Join(examplePathParts, string(os.PathSeparator)))
		} else {
			changed := false
			if parts[0] != "textures" {
				parts[0] = "textures"
				changed = true
			}
			if parts[1] != "objects" {
				parts[1] = "objects"
				changed = true
			}
			if parts[len(parts)-1] != "object.png" {
				parts[len(parts)-1] = "object.png"
				changed = true
			}
			examplePathParts = parts
			if changed {
				examplePathEntry.SetText(strings.Join(examplePathParts, string(os.PathSeparator)))
			}
		}
		updateTagsMap()
	}

	bindings.Listen(boundPrefixSplitSeparator, func(separator string) {
		if prefixSplitMode && lastSplitSeperator != "" && prefixSplitSeparator != "" {
			newParts := slices.Collect(utils.Map(slices.Values(examplePathParts[2:len(examplePathParts)-1]), func(part string) string {
				return strings.ReplaceAll(part, lastSplitSeperator, prefixSplitSeparator)
			}))

			examplePathParts = slices.Concat([]string{"textures", "objects"}, newParts, []string{"object.png"})
			examplePathEntry.SetText(strings.Join(examplePathParts, string(os.PathSeparator)))
			updateTagsMap()
			lastSplitSeperator = prefixSplitSeparator
		}
	})

	bindings.AddListenerToAll(func() {
		if !prefixSplitMode &&
			lastDelimiter[0] != "" && lastDelimiter[1] != "" &&
			tagSetPrefixDelimiter[0] != "" && tagSetPrefixDelimiter[1] != "" {

			newParts := slices.Collect(utils.Map(slices.Values(examplePathParts[2:len(examplePathParts)-1]), func(part string) string {
				return strings.ReplaceAll(strings.ReplaceAll(part, lastDelimiter[0], tagSetPrefixDelimiter[0]), lastDelimiter[1], tagSetPrefixDelimiter[1])
			}))
			examplePathParts = slices.Concat([]string{"textures", "objects"}, newParts, []string{"object.png"})
			examplePathEntry.SetText(strings.Join(examplePathParts, string(os.PathSeparator)))
			updateTagsMap()
			lastDelimiter = [2]string{tagSetPrefixDelimiter[0], tagSetPrefixDelimiter[1]}
		}
	},
		boundPFDStart,
		boundPFDStop,
	)

	examplePathContainer := container.New(
		layout.NewFormLayout(),
		examplePathLbl, examplePathEntry,
	)

	listsContainer := container.New(
		xlayout.NewHPortion([]float64{50, 0.1, 50}),
		layouts.NewBottomExpandVBox(
			container.NewStack(
				&canvas.Rectangle{
					FillColor:    theme.Color(theme.ColorNameHeaderBackground),
					CornerRadius: 4,
				},
				container.NewPadded(
					widget.NewLabel(lang.X("pathGen.exampleTags.label", "Example Tags")),
				),
			),
			container.NewStack(
				&canvas.Rectangle{
					FillColor:    theme.Color(theme.ColorNameInputBackground),
					CornerRadius: 4,
				},
				tagsList,
			),
		),
		widget.NewSeparator(),
		layouts.NewBottomExpandVBox(
			container.NewStack(
				&canvas.Rectangle{
					FillColor:    theme.Color(theme.ColorNameHeaderBackground),
					CornerRadius: 4,
				},
				container.NewPadded(
					widget.NewLabel(lang.X("pathGen.exampleSets.label", "Example sets tag is in")),
				),
			),
			container.NewStack(
				&canvas.Rectangle{
					FillColor:    theme.Color(theme.ColorNameInputBackground),
					CornerRadius: 4,
				},
				setsList,
			),
		),
	)

	useGlobalTagCheck := widget.NewCheckWithData(
		lang.X("pathGen.useGlobalTagCheck.label", "Add all tags to a global tag set"),
		boundBuildGlobalTagSet,
	)
	globalTagSetEntry := widget.NewEntryWithData(boundGlobalTagSet)
	bindings.Listen(boundBuildGlobalTagSet, func(checked bool) {
		if checked {
			globalTagSetEntry.Enable()
		} else {
			globalTagSetEntry.Disable()
		}
	})
	globalTagSetEntry.Validator = nil
	globalTagSetLbl := widget.NewLabel(
		lang.X("pathGen.globalTagSet.label", "Global Tag Set Name"),
	)

	buildFromPrefixCheck := widget.NewCheckWithData(
		lang.X("pathGen.buildFromPrefixCheck.label", "Build tag sets from prefixes"),
		boundBuildTagSetsFromPrefix,
	)

	globalTagContainer := container.NewVBox(
		useGlobalTagCheck,
		container.New(
			layout.NewFormLayout(),
			globalTagSetLbl, globalTagSetEntry,
		),
	)

	prefixSplitModeCheck := widget.NewCheckWithData(
		lang.X("pathGen.prefixSplitModeCheck.label", "Use a separator instead of a delimited prefix"),
		boundPrefixSplitMode,
	)

	prefixDelimStartEntry := widget.NewEntryWithData(boundPFDStart)
	prefixDelimStartEntry.Validator = nil
	prefixDelimStopEntry := widget.NewEntryWithData(boundPFDStop)
	prefixDelimStopEntry.Validator = nil
	prefixDelimStartLbl := widget.NewLabel(
		lang.X("pathGen.prefixDelimStart.label", "Start Delimiter"),
	)
	prefixDelimStopLbl := widget.NewLabel(
		lang.X("pathGen.prefixDelimStop.label", "Stop Delimiter"),
	)

	prefixDelim := container.New(
		xlayout.NewHPortion([]float64{50, 50}),
		container.New(
			layout.NewFormLayout(),
			prefixDelimStartLbl, prefixDelimStartEntry,
		),
		container.New(
			layout.NewFormLayout(),
			prefixDelimStopLbl, prefixDelimStopEntry,
		),
	)

	prefixSepEntry := widget.NewEntryWithData(boundPrefixSplitSeparator)
	prefixSepEntry.Validator = nil
	prefixSplitLbl := widget.NewLabel(
		lang.X("pathGen.prefixSplit.label", "Prefix Separator"),
	)

	prefixSplit := container.New(
		layout.NewFormLayout(),
		prefixSplitLbl, prefixSepEntry,
	)
	prefixSplit.Hide()

	bindings.Listen(boundPrefixSplitMode, func(checked bool) {
		var newParts []string
		if checked {
			newParts = slices.Collect(utils.Map2(maps.All(
				ddpackage.NewGenerateTags(&ddpackage.GenerateTagsOptions{
					BuildGlobalTagSet:      buildGlobalTagSet,
					GlobalTagSet:           globalTagSet,
					BuildTagSetsFromPrefix: buildTagSetFrpmPrefix,
					PrefixSplitMode:        false,
					TagSetPrefrixDelimiter: tagSetPrefixDelimiter,
					StripTagSetPrefix:      stripTagSetPrefix,
					StripExtraPrefix:       stripExtraPrefix,
				}).
					TagsFromPath(strings.Join(examplePathParts, "/")),
			), func(tag string, sets *structures.Set[string]) string {
				return strings.Join(
					slices.Concat(
						slices.Collect(utils.Filter(sets.Values(), func(set string) bool {
							return set != globalTagSet
						})),
						[]string{tag},
					),
					prefixSplitSeparator,
				)
			}))
			prefixDelim.Hide()
			prefixSplit.Show()
		} else {
			newParts = slices.Collect(utils.Map2(maps.All(
				ddpackage.NewGenerateTags(&ddpackage.GenerateTagsOptions{
					BuildGlobalTagSet:      buildGlobalTagSet,
					GlobalTagSet:           globalTagSet,
					BuildTagSetsFromPrefix: buildTagSetFrpmPrefix,
					PrefixSplitMode:        true,
					TagSetPrefrixDelimiter: [2]string{prefixSplitSeparator, ""},
					StripTagSetPrefix:      stripTagSetPrefix,
					StripExtraPrefix:       stripExtraPrefix,
				}).
					TagsFromPath(strings.Join(examplePathParts, "/")),
			), func(tag string, sets *structures.Set[string]) string {
				return strings.Join(
					slices.Collect(utils.Map(
						utils.Filter(sets.Values(), func(set string) bool {
							return set != globalTagSet
						}),
						func(set string) string {
							return tagSetPrefixDelimiter[0] + set + tagSetPrefixDelimiter[1]
						})),
					"",
				) + tag
			}))
			prefixDelim.Show()
			prefixSplit.Hide()
		}
		examplePathParts = slices.Concat([]string{"textures", "objects"}, newParts, []string{"object.png"})
		examplePathEntry.SetText(strings.Join(examplePathParts, string(os.PathSeparator)))
		updateTagsMap()
	})

	stripPrefixFromTagCheck := widget.NewCheckWithData(
		lang.X("pathGen.stripPrefixFromTagCheck.label", "Strip tag set prefix from generated tag"),
		boundStripTagSetPrefix,
	)

	prefixContainer := container.NewVBox(
		prefixSplitModeCheck,
		prefixDelim,
		prefixSplit,
		stripPrefixFromTagCheck,
	)

	bindings.Listen(boundBuildTagSetsFromPrefix, func(checked bool) {
		if checked {
			prefixContainer.Show()
		} else {
			prefixContainer.Hide()
		}
	})

	stripExtraPrefixEntry := widget.NewEntryWithData(boundStripExtraPrefix)
	stripExtraPrefixEntry.Validator = nil
	stripExtraPrefixLbl := widget.NewLabel(
		lang.X("pathGen.stripExtraPrefix", "Prefix to strip from the generated tags"),
	)

	stripExtraContainer := container.New(
		layout.NewFormLayout(),
		stripExtraPrefixLbl, stripExtraPrefixEntry,
	)

	var genTagsDlg *dialog.CustomDialog

	generateBtn := widget.NewButtonWithIcon(
		lang.X("pathGen.generateBtl.label", "Generate"),
		theme.ConfirmIcon(),
		func() {
			log.Info("Generating tags...")
			progressVal := binding.NewFloat()
			progressBar := widget.NewProgressBarWithData(progressVal)

			progressDlg := dialog.NewCustomWithoutButtons(
				lang.X("pathGen.tagProgressDlg.title", "Generating Tags ..."),
				container.NewVBox(progressBar),
				a.window,
			)
			progressDlg.Show()
			a.pkg.GenerateTagsProgress(generator, func(p float64) {
				progressVal.Set(p)
			})
			progressDlg.Hide()
			doneDlg := dialog.NewInformation(
				lang.X("pathGen.doneDialog.title", "Tags Generated"),
				lang.X("pathGen.doneDialog.msg", "Tags have finished generating."),
				a.window,
			)
			doneDlg.SetOnClosed(func() {
				genTagsDlg.Hide()
			})
			doneDlg.Show()
		},
	)

	example := layouts.NewBottomExpandVBox(
		examplePathContainer,
		listsContainer,
	)

	controls := container.NewVBox(
		globalTagContainer,
		buildFromPrefixCheck,
		prefixContainer,
		stripExtraContainer,
		generateBtn,
	)

	content := container.NewPadded(
		layouts.NewTopExpandVBox(
			example,
			controls,
		),
	)

	genTagsDlg = dialog.NewCustom(
		lang.X("pathGen.dialog.title", "Generate Tags"),
		lang.X("pathGen.dialog.dismiss", "Close"),
		content,
		a.window,
	)

	genTagsDlg.Resize(
		fyne.NewSize(
			fyne.Min(a.window.Canvas().Size().Width, 800),
			fyne.Min(a.window.Canvas().Size().Height, 760),
		),
	)

	return genTagsDlg
}
