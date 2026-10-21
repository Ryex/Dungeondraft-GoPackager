package gui

import (
	"errors"
	"fmt"
	"image/color"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/davecgh/go-spew/spew"

	"github.com/ryex/dungeondraft-gopackager/internal/gui/bindings"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/lang"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/layouts"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/named"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/resource_tree"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/widgets"
	"github.com/ryex/dungeondraft-gopackager/internal/utils"

	"github.com/ryex/dungeondraft-gopackager/pkg/ddimage"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
	ddresource_filter "github.com/ryex/dungeondraft-gopackager/pkg/ddpackage/filter"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"

	ddcolor "github.com/ryex/dungeondraft-gopackager/pkg/structures/color"
	log "github.com/sirupsen/logrus"
)

// The names of the parts of the package content.
//
// The parts of the interface are named where they are created, so that they
// can be looked up by name later, e.g. by the specification tests.
const (
	nameResourceTree    = "resourceTree"
	nameResourcePane    = "resourcePane"
	nameSplitPane       = "splitPane"
	nameFilterEntry     = "filterEntry"
	nameDisplayByRadio  = "displayByRadio"
	nameInfoTabs        = "infoTabs"
	nameFilePreview     = "filePreview"
	nameTagInfo         = "tagInfo"
	nameMetadataPane    = "metadataPane"
	nameMetadataForm    = "metadataForm"
	nameBulkOpsPane     = "bulkOpsPane"
	nameThumbnailToggle = "thumbnailToggle"
	nameImagePreview    = "imagePreview"
	nameTextPreview     = "textPreview"
	nameTooLargeLabel   = "tooLargeLabel"
	nameTagList         = "tagList"
)

func (a *App) buildPackageTreeAndInfoPane(editable bool) *named.Named {
	tree, filter, displayByTag := a.buildPackageTree(editable)

	filterEntry := widgets.NewToolTipEntryWithData(filter)
	filterEntry.SetPlaceHolder(lang.X("tree.filter.placeholder.resource", "Filter with glob (e.g. */objects/**)"))
	filterErrorText := widgets.NewThemedText(lang.X("tree.filter.error", "Bad glob syntax"), theme.ColorNameError)
	filterErrorText.Hide()
	filterEntry.Validator = func(s string) error {
		_, err := ddresource_filter.ParseResourceFilter(s)
		if err != nil {
			filterErrorText.Text.Text = lang.X("tree.filter.error", "Bad filter syntax") + ": " + err.Error()
			filterErrorText.Show()
		} else {
			filterErrorText.Text.Text = ""
			filterErrorText.Hide()
		}
		return err
	}

	filterEntry.SetToolTip(lang.X("tree.filter.toolTip.glob", "Use Glob syntax to match resource paths"))

	displayByLbl := widget.NewLabel(lang.X("tree.displayBy.label", "Display By"))

	byResourceOption := lang.X("tree.displayby.resource", "Resource")
	byTagOption := lang.X("tree.displayby.tag", "Tag")

	displayByRadio := widget.NewRadioGroup([]string{byResourceOption, byTagOption}, func(selected string) {
		if selected == byResourceOption {
			displayByTag.Set(false)
			filterEntry.SetPlaceHolder(lang.X("tree.filter.placeholder.resource", "Filter with glob (e.g. */objects/**)"))
			filterEntry.SetToolTip(lang.X("tree.filter.toolTip.glob", "Use Glob syntax to match resource paths"))
		} else {
			displayByTag.Set(true)
			filterEntry.SetPlaceHolder(lang.X("tree.filter.placeholder.tags", "Filter by tag name"))
			filterEntry.SetToolTip(lang.X(
				"tree.filter.toolTip.tag",
				"Filter by a set of tags. Separate tags with spaces to use multiple.\n"+
					"If a tag contains spaces surround it with \"\".\n"+
					"By default filters use OR (filter includes any resource that"+
					" has at least one of the filter tags).\n"+
					"Start the filter with a single \"&\" to filter by AND (e.g `& aTag \"another Tag\" tagC`).\n"+
					"By Default tags use a fuzzy match, prefix a tag with a `%` to make it match exactly (e.g \" `%exactly_this_tag`\").",
			))
		}
	})
	displayByRadio.Required = true
	displayByRadio.Horizontal = true
	displayByRadio.SetSelected(byResourceOption)

	displayByContainer := layouts.NewRightExpandHBox(
		displayByLbl, named.New(nameDisplayByRadio, displayByRadio),
	)

	leftSplit := layouts.NewTopExpandVBox(
		layouts.NewBottomExpandVBox(
			displayByContainer,
			layouts.NewRightExpandHBox(
				widget.NewLabel(lang.X("tree.label", "Resources")),
				named.New(nameFilterEntry, filterEntry),
			),
			container.NewStack(
				widgets.NewThemedRect(theme.ColorNameInputBackground, 4),
				container.NewPadded(named.New(nameResourceTree, tree)),
			),
		),
	)

	defaultPreview := container.NewCenter(
		widget.NewLabel(lang.X("preview.defaultText", "Select a resource")),
	)

	rightSplit := container.NewStack(defaultPreview)

	tree.OnSelectionChanged = func() {
		selected := tree.GetSelectionAndChildData()
		content := func() fyne.CanvasObject {
			if len(selected) == 0 {
				return defaultPreview
			} else if len(selected) == 1 {
				switch data := selected[0].(type) {
				case *resource_tree.ResourceFileInfoNode:
					return a.buildInfoPane(data.Info, editable)
				}
			}
			selectedInfos := slices.Collect(utils.Filter(utils.Map(slices.Values(selected), func(data resource_tree.ResourceTreeNodeData) *structures.FileInfo {
				if info, ok := data.(*resource_tree.ResourceFileInfoNode); ok {
					return info.Info
				}
				return nil
			}), func(info *structures.FileInfo) bool {
				return info != nil
			}))

			return a.buildBulkOpsPane(selectedInfos, editable)
		}()
		rightSplit.RemoveAll()
		rightSplit.Add(content)
		rightSplit.Refresh()

		// the parts of the information pane are new objects, so the cache
		// of the named parts has to forget the old ones
		if a.ui != nil {
			a.ui.Reset()
		}
	}

	split := container.NewPadded(container.NewHSplit(
		container.NewPadded(leftSplit),
		container.NewPadded(named.New(nameResourcePane, rightSplit)),
	))

	return named.New(nameSplitPane, split)
}

func (a *App) buildPackageTree(editable bool) (*resource_tree.ResourceTree, binding.String, binding.Bool) {
	filter := ""
	boundFilter := binding.BindString(&filter)
	byTag := false
	boundByTag := binding.BindBool(&byTag)

	resourceTree := resource_tree.NewResourceTree(a.pkg, a.window)

	refreshTree := func() {
		resourceTree.RefreshTree()
		log.Debug("Refresh Done")
		resourceTree.OpenBranch(resource_tree.PackageTreeRootID)
	}

	boundFilter.AddListener(binding.NewDataListener(func() {
		f, err := ddresource_filter.ParseResourceFilter(filter)
		if err != nil {
			return
		}
		log.Debugf("Using filter: %s", f)
		resourceTree.SetFilter(f)
		resourceTree.OpenBranch(resource_tree.PackageTreeRootID)
	}))

	boundByTag.AddListener(binding.NewDataListener(func() {
		mode := resourceTree.GetMode()
		var wantedMode resource_tree.ResourceTreeMode
		if byTag {
			wantedMode = resource_tree.ResourceTreeTagMode
		} else {
			wantedMode = resource_tree.ResourceTreePathMode
		}
		if mode != wantedMode {
			resourceTree.SetMode(wantedMode)
			resourceTree.OpenBranch(resource_tree.PackageTreeRootID)
		}
	}))

	bindings.AddListenerToAll(
		refreshTree,
		a.packageUpdated,
	)

	return resourceTree, boundFilter, boundByTag
}

func (a *App) buildInfoPane(info *structures.FileInfo, editable bool) fyne.CanvasObject {
	tabs := make(map[string]*container.TabItem, 3)
	tabs["Resource"] = container.NewTabItemWithIcon(
		lang.X("preview.tab.resource", "Resource"),
		theme.FileIcon(),
		named.New(nameFilePreview, a.buildFilePreview(info)),
	)

	tabsContainer := container.NewAppTabs(tabs["Resource"])

	if info.IsTaggable() {
		tabs["Tags"] = container.NewTabItemWithIcon(
			lang.X("preview.tab.tags", "Tags"),
			theme.ListIcon(),
			named.New(nameTagInfo, a.buildTagInfo(info, editable)),
		)
		tabsContainer.Append(tabs["Tags"])
	}

	if info.ShouldHaveMetadata() {
		tabs["Settings"] = container.NewTabItemWithIcon(
			lang.X("preview.tab.metadata", "Settings"),
			theme.ColorPaletteIcon(),
			named.New(nameMetadataPane, a.buildMetadataPane(info, editable)),
		)
		tabsContainer.Append(tabs["Settings"])
	}

	tabsContainer.SetTabLocation(container.TabLocationTop)

	if tabs[a.lastSelectedTab] != nil {
		tabsContainer.Select(tabs[a.lastSelectedTab])
	}

	tabsContainer.OnSelected = func(ti *container.TabItem) {
		for key := range tabs {
			if tabs[key] == ti {
				a.lastSelectedTab = key
			}
		}
	}

	return container.NewBorder(
		nil, nil, nil, nil,
		named.New(nameInfoTabs, tabsContainer),
	)
}

func (a *App) buildFilePreview(info *structures.FileInfo) fyne.CanvasObject {
	fileData, err := a.pkg.LoadResource(info.ResPath)
	if err != nil {
		log.WithError(err).Errorf("failed to read image data for %s", info.ResPath)
		return widget.NewLabel(fmt.Sprintf("Failed to read image data for %s", info.ResPath))
	}

	showThumbnail := binding.BindPreferenceBool("showThumbnails", a.app.Preferences())
	thumbnailToggle := widgets.NewToggleWithData(showThumbnail)
	thumbnailLbl := widget.NewLabel(lang.X("preview.thumbnail.toggle", "Show Thumbnail"))
	thumbToggle := named.New(nameThumbnailToggle, layouts.NewLeftExpandHBox(thumbnailLbl, thumbnailToggle))
	if !info.IsTexture() {
		thumbToggle.Hide()
	}

	resPathLabel := widget.NewLabel(lang.X("preview.path.label", "Resource Path"))
	resPath := widgets.NewEntryWCopyBtn()
	resPath.Disable()
	resPath.SetText(info.ResPath)

	resMd5Label := widget.NewLabel(lang.X("preview.md5.label", "Md5 Hash"))
	resMd5 := widgets.NewEntryWCopyBtn()
	resMd5.Disable()

	fsPathLabel := widget.NewLabel(lang.X("preview.fspath.label", "Filesystem Path"))
	fsPath := widgets.NewEntryWCopyBtn()
	fsPath.Disable()
	fsPath.SetText(info.Path)

	labelCont := container.NewVBox(resPathLabel, fsPathLabel, resMd5Label)
	entryCont := container.NewVBox(resPath, fsPath, resMd5)
	infoCont := container.NewBorder(nil, nil, labelCont, nil, entryCont)

	// the md5 of the resource is computed in the background when it is not
	// known yet, so the update of the entry is tracked as background work
	// that touches the interface
	a.bgWG.Add(1)
	a.pkg.GetOrUpdateResourceMd5(info, func(md5 string, err error) {
		defer a.bgWG.Done()

		if err != nil {
			log.WithError(err).Errorf("Failed to update hash for %s", info.ResPath)
		}
		// the entry may only be touched on the Fyne goroutine
		fyne.Do(func() {
			resMd5.SetText(md5)
		})
	})

	tooLarge := named.New(nameTooLargeLabel, container.NewCenter(
		widget.NewLabel(lang.X("preview.tooLarge", "This file is too large!\nOpen it in a text editor.")),
	))

	bg := widgets.NewThemedRect(theme.ColorNameInputBackground, 4)

	if !ddimage.PathIsSupportedImage(info.RelPath) {
		textContent := string(fileData)
		if len(strings.Split(textContent, "\n")) > 200 {
			return container.NewPadded(
				layouts.NewBottomExpandVBox(
					infoCont,
					container.NewStack(
						bg, tooLarge,
					),
				),
			)
		}
		textEntry := widget.NewMultiLineEntry()
		textEntry.Text = textContent
		copyBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
			a.app.Clipboard().SetContent(string(fileData))
		})
		content := container.NewPadded(
			layouts.NewBottomExpandVBox(
				infoCont,
				container.NewStack(
					bg,
					container.NewPadded(named.New(nameTextPreview, textEntry)),
					container.NewPadded(
						container.NewVBox(
							container.NewHBox(layout.NewSpacer(), copyBtn),
							layout.NewSpacer(),
						),
					),
				),
			),
		)
		return content
	}

	img, _, err := ddimage.BytesToImage(fileData)
	if err != nil {
		log.WithError(err).Errorf("failed to decode image for %s", info.ResPath)
		content := container.NewCenter(
			widget.NewLabel(
				fmt.Sprintf(
					"Failed to decode image for %s\n%s",
					info.ResPath, err.Error(),
				),
			),
		)
		return content
	}

	thumbnailErrString := lang.X("preview.noThumbnail", "Thumbnail not generated.")
	thumbnailErr := binding.BindString(&thumbnailErrString)
	thumbnailErrObj := container.NewCenter(
		container.NewStack(
			widgets.NewThemedRect(theme.ColorNameBackground, 8),
			widget.NewLabelWithData(thumbnailErr),
		),
	)

	var thumbnail fyne.CanvasObject = thumbnailErrObj
	if info.ThumbnailResPath != "" {
		thumbnailData, thumbErr := a.pkg.LoadResource(info.ThumbnailResPath)
		if thumbErr != nil {
			if errors.Is(thumbErr, ddpackage.ErrResourceNotFound) {
				thumbnailErr.Set(lang.X("preview.noThumbnail", "Thumbnail not generated."))
			} else {
				thumbnailErr.Set(lang.X("preview.thumbnailError", "Error loading thumbnail.\n{{.Error}}", map[string]any{"Error": thumbErr.Error()}))
			}
		} else {
			thumb, _, thumbErr := ddimage.BytesToImage(thumbnailData)
			if thumbErr != nil {
				thumbnailErr.Set(lang.X("preview.thumbnailError", "Error loading thumbnail.\n{{.Error}}", map[string]any{"Error": thumbErr.Error()}))
			} else {
				thumbW := canvas.NewImageFromImage(thumb)
				thumbW.FillMode = canvas.ImageFillOriginal
				thumbnail = thumbW
			}
		}
	}
	thumbnail.Hide()

	log.Infof("loaded image for %s", info.ResPath)
	previewImage := canvas.NewImageFromImage(img)
	previewImage.FillMode = canvas.ImageFillOriginal
	previewImage.ScaleMode = canvas.ImageScaleFastest
	imgW := named.New(nameImagePreview, previewImage)

	imgContent := container.NewScroll(
		container.NewStack(
			container.NewCenter(imgW),
			container.NewCenter(thumbnail),
		),
	)

	bindings.Listen(showThumbnail, func(show bool) {
		if show && info.IsTexture() {
			imgW.Hide()
			thumbnail.Show()
		} else {
			thumbnail.Hide()
			imgW.Show()
		}
		imgContent.Refresh()
	})

	content := container.NewPadded(layouts.NewBottomExpandVBox(
		thumbToggle,
		infoCont,
		container.NewStack(
			canvas.NewRasterWithPixels(func(x, y, w, h int) color.Color {
				if ((x/8)+(y/8))%2 == 0 {
					return color.NRGBA{90, 90, 90, 255}
				}
				return color.NRGBA{160, 160, 160, 255}
			}),
			container.NewPadded(imgContent),
		),
	))
	return content
}

func (a *App) buildTagInfo(info *structures.FileInfo, editable bool) fyne.CanvasObject {
	var tags []string
	boundTags := binding.BindStringList(&tags)
	updateTags := func() {
		tags = a.pkg.Tags().TagsFor(info.RelPath).AsSlice()
		slices.Sort(tags)
		boundTags.Reload()
		log.Infof("tags for %s: %s", info.ResPath, spew.Sdump(tags))
	}
	updateTags()

	tagsList := widget.NewListWithData(
		boundTags,
		func() fyne.CanvasObject {
			return layouts.NewLeftExpandHBox(
				widget.NewLabel("template"),
				container.NewPadded(widgets.NewToolTipButtonWithIcon("", theme.DeleteIcon(), nil)),
			)
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			c := co.(*fyne.Container)
			l := c.Objects[0].(*widget.Label)
			l.Bind(di.(binding.String))
			btn := c.Objects[1].(*fyne.Container).Objects[0].(*widgets.ToolTipButton)
			if editable {
				tag, err := di.(binding.String).Get()
				if err != nil {
					log.WithError(err).Errorf("failed to get tag in del tag btn for %s", info.RelPath)
				} else {
					btn.OnTapped = func() {
						a.pkg.Tags().Untag(tag, info.RelPath)
						updateTags()
						a.saveUnpackedTags()
					}
					btn.SetToolTip(lang.X(
						"resource.tag.remove.tooltip",
						"Remove the '{{.Tag}}' tag from this object",
						map[string]string{
							"Tag": tag,
						},
					))
				}
			} else {
				btn.Disable()
				btn.Hide()
			}
		},
	)

	content := layouts.NewTopExpandVBox(
		layouts.NewBottomExpandVBox(
			container.NewStack(
				widgets.NewThemedRect(theme.ColorNameHeaderBackground, 4),
				layouts.NewRightExpandHBox(
					container.NewCenter(
						widget.NewLabel(lang.X(
							"preview.tab.tags.label",
							"Tags for",
						)),
					),
					container.NewPadded(
						container.NewStack(
							widgets.NewThemedRect(theme.ColorNameInputBackground, 4),
							container.NewPadded(
								container.NewHScroll(
									widgets.NewThemedText(info.ResPath, theme.ColorNameForeground),
								),
							),
						),
					),
				),
			),
			container.NewStack(
				widgets.NewThemedRect(theme.ColorNameInputBackground, 4),
				container.NewPadded(
					named.New(nameTagList, tagsList),
				),
			),
		),
	)

	if editable {
		tagSelector := widget.NewSelectEntry(a.pkg.Tags().AllTags())
		addBtn := widget.NewButtonWithIcon(
			lang.X("tags.addBtn.text", "Add"),
			theme.ContentAddIcon(),
			func() {
				if tagSelector.Text == "" {
					return
				}
				a.pkg.Tags().Tag(tagSelector.Text, info.RelPath)
				updateTags()
				a.saveUnpackedTags()
				tagSelector.SetText("")
				tagSelector.SetOptions(a.pkg.Tags().AllTags())
			},
		)
		content.Add(layouts.NewLeftExpandHBox(tagSelector, addBtn))
	}

	return container.NewPadded(content)
}

func (a *App) saveUnpackedTags() {
	if a.tagSaveTimer != nil {
		a.tagSaveTimer.Stop()
		a.tagSaveTimer = nil
	}
	a.tagSaveTimer = time.AfterFunc(500*time.Millisecond, func() {
		a.tagSaveTimer = nil
		err := a.pkg.SaveUnpackedTags()
		if err != nil {
			a.showErrorDialog(err)
		}
	})
}

func (a *App) buildMetadataPane(info *structures.FileInfo, editable bool) fyne.CanvasObject {
	content := container.NewPadded()

	metaContent := func() fyne.CanvasObject {
		if info.IsWall() {
			metaPath := info.MetadataPath

			defaultColor := color.NRGBA{255, 255, 255, 255}
			wallData := a.pkg.Walls()
			metaData, ok := wallData[metaPath]
			if ok {
				defaultColor = metaData.Color.ToColor()
			} else {
				log.WithField("res", info.ResPath).
					WithField("metaRes", metaPath).
					Warn("Missing wall metadata")
			}

			colorLbl := widget.NewLabel(lang.X("metadata.color.label", "Color"))
			colorRect := widgets.NewTappableRect(defaultColor, 4)
			colorRect.SetMinSize(fyne.NewSize(48, 32))
			if editable {
				colorRect.OnTapped = func(_ *fyne.PointEvent) {
					dlg := dialog.NewColorPicker(
						lang.X("metadata.colorPickDialog.title", "Pick a default color"),
						"",
						func(c color.Color) {
							colorRect.SetColor(c)
							if wallData != nil {
								data, ok := wallData[metaPath]
								if !ok {
									wallData[metaPath] = structures.PackageWall{
										Path:  info.RelPath,
										Color: ddcolor.FromColor(c),
									}
								} else {
									data.Color = ddcolor.FromColor(c)
									wallData[metaPath] = data
								}
								a.saveWallMetadata(metaPath)
							}
						},
						a.window,
					)
					dlg.Advanced = true
					dlg.SetColor(colorRect.GetColor())
					dlg.Show()
				}
			}
			form := named.New(nameMetadataForm, container.New(
				layout.NewFormLayout(),
				colorLbl, colorRect,
			))
			return form

		} else if info.IsTileset() {
			metaPath := info.MetadataPath

			defaultColor := color.NRGBA{255, 255, 255, 255}
			tilesetName := ""
			tilesetType := structures.TilesetNormal

			tilesetData := a.pkg.Tilesets()
			metaData, ok := tilesetData[metaPath]
			if ok {
				defaultColor = metaData.Color.ToColor()
				tilesetName = metaData.Name
				tilesetType = metaData.Type
			} else {
				log.WithField("res", info.ResPath).
					WithField("metaRes", metaPath).
					Warn("Missing tileset metadata")
			}

			colorLbl := widget.NewLabel(lang.X("metadata.color.label", "Color"))
			colorRect := widgets.NewTappableRect(defaultColor, 4)
			colorRect.SetMinSize(fyne.NewSize(48, 32))
			if editable {
				colorRect.OnTapped = func(_ *fyne.PointEvent) {
					dlg := dialog.NewColorPicker(
						lang.X("metadata.colorPickDialog.title", "Pick a default color"),
						"",
						func(c color.Color) {
							colorRect.SetColor(c)
							if tilesetData != nil {
								data, ok := tilesetData[metaPath]
								if !ok {
									tilesetData[metaPath] = structures.PackageTileset{
										Path:  info.RelPath,
										Name:  "",
										Color: ddcolor.FromColor(c),
										Type:  structures.TilesetNormal,
									}
								} else {
									data.Color = ddcolor.FromColor(c)
									tilesetData[metaPath] = data
								}
								a.saveTilesetMetadata(metaPath)
							}
						},
						a.window,
					)
					dlg.Advanced = true
					dlg.SetColor(colorRect.GetColor())
					dlg.Show()
				}
			}

			nameLbl := widget.NewLabel(lang.X("metadata.name.label", "Name"))
			nameEntry := widget.NewEntry()
			nameEntry.SetText(tilesetName)
			nameEntry.OnChanged = func(s string) {
				data, ok := tilesetData[metaPath]
				if !ok {
					tilesetData[metaPath] = structures.PackageTileset{
						Path:  info.RelPath,
						Name:  s,
						Color: ddcolor.FromColor(defaultColor),
						Type:  structures.TilesetNormal,
					}
				} else {
					data.Name = s
					tilesetData[metaPath] = data
				}
				a.saveTilesetMetadata(metaPath)
			}

			tilesetTypeLbl := widget.NewLabel(lang.X("metadata.tilesetType.label", "Tileset Type"))
			tilesetTypeSelector := widget.NewSelect(
				[]string{
					string(structures.TilesetNormal),
					string(structures.TilesetCustomColor),
				},
				nil,
			)
			tilesetTypeSelector.SetSelected(string(tilesetType))
			tilesetTypeSelector.OnChanged = func(s string) {
				var typ structures.TilesetType
				switch s {
				case string(structures.TilesetNormal):
					typ = structures.TilesetNormal
				case string(structures.TilesetCustomColor):
					typ = structures.TilesetCustomColor
				}
				data, ok := tilesetData[metaPath]
				if !ok {
					tilesetData[metaPath] = structures.PackageTileset{
						Path:  info.RelPath,
						Name:  "",
						Color: ddcolor.FromColor(defaultColor),
						Type:  typ,
					}
				} else {
					data.Type = typ
					tilesetData[metaPath] = data
				}
				a.saveTilesetMetadata(metaPath)
			}

			if !editable {
				nameEntry.Disable()
				tilesetTypeSelector.Disable()
			}

			form := named.New(nameMetadataForm, container.New(
				layout.NewFormLayout(),
				nameLbl, nameEntry,
				tilesetTypeLbl, tilesetTypeSelector,
				colorLbl, colorRect,
			))
			return form

		}
		return nil
	}()

	if metaContent != nil {
		content.Add(metaContent)
	}

	return content
}

func (a *App) saveWallMetadata(metaPath string) {
	timer := a.resSaveTimers[metaPath]
	if timer != nil {
		timer.Stop()
	}
	a.resSaveTimers[metaPath] = time.AfterFunc(500*time.Millisecond, func() {
		a.resSaveTimers[metaPath] = nil
		log.Infof("save timer called for %s", metaPath)
		err := a.pkg.SaveUnpackedWall(metaPath)
		if err != nil {
			a.showErrorDialog(err)
		}
	})
}

func (a *App) saveTilesetMetadata(metaPath string) {
	timer := a.resSaveTimers[metaPath]
	if timer != nil {
		timer.Stop()
	}
	a.resSaveTimers[metaPath] = time.AfterFunc(500*time.Millisecond, func() {
		a.resSaveTimers[metaPath] = nil
		log.Infof("save timer called for %s", metaPath)
		err := a.pkg.SaveUnpackedTileset(metaPath)
		if err != nil {
			a.showErrorDialog(err)
		}
	})
}

func (a *App) buildBulkOpsPane(infos []*structures.FileInfo, editable bool) fyne.CanvasObject {
	// TODO: BUILD
	return named.New(nameBulkOpsPane, container.NewBorder(nil, nil, nil, nil))
}
