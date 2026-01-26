package gui

import (
	"errors"
	"fmt"
	"image/color"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/davecgh/go-spew/spew"

	"github.com/ryex/dungeondraft-gopackager/internal/gui/bindings"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/lang"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/layouts"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/widgets"

	"github.com/ryex/dungeondraft-gopackager/internal/utils"

	"github.com/ryex/dungeondraft-gopackager/pkg/ddimage"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"

	ddcolor "github.com/ryex/dungeondraft-gopackager/pkg/structures/color"
	log "github.com/sirupsen/logrus"
)

func (a *App) buildPackageTreeAndInfoPane(editable bool) fyne.CanvasObject {
	tree, filter, treeSelected, displayByTag, nodeTree := a.buildPackageTree(editable)

	filterEntry := widgets.NewToolTipEntryWithData(filter)
	filterEntry.SetPlaceHolder(lang.X("tree.filter.placeholder.resource", "Filter with glob (e.g. */objects/**)"))
	filterErrorText := widgets.NewThemedText(lang.X("tree.filter.error", "Bad glob syntax"), theme.ColorNameError)
	filterErrorText.Hide()
	filterEntry.Validator = func(s string) error {
		byTag, err := displayByTag.Get()
		if err != nil {
			return err
		}
		if byTag {
			filterErrorText.Hide()
		} else {
			_, err := structures.GlobToRelPathRegexp(s)
			filterErrorText.Text.Text = lang.X("tree.filter.error", "Bad glob syntax")
			if err != nil {
				filterErrorText.Show()
			} else {
				filterErrorText.Hide()
			}
		}
		return err
	}

	filterEntry.SetToolTip(lang.X("tree.filter.toolTip.glob", "Use Glob syntax to match resource paths"))

	displayByLbl := widget.NewLabel(lang.X("tree.displayBy.label", "Display By"))

	byResouceOption := lang.X("tree.displayby.resource", "Resource")
	byTagOption := lang.X("tree.displayby.tag", "Tag")

	displayByRadio := widget.NewRadioGroup([]string{byResouceOption, byTagOption}, func(selected string) {
		if selected == byResouceOption {
			displayByTag.Set(false)
			filterEntry.SetPlaceHolder(lang.X("tree.filter.placeholder.resource", "Filter with glob (e.g. */objects/**)"))
			filterEntry.SetToolTip(lang.X("tree.filter.toolTip.glob", "Use Glob syntax to match resource paths"))
		} else {
			displayByTag.Set(true)
			filterEntry.SetPlaceHolder(lang.X("tree.filter.placeholder.tags", "Filter by tag name"))
			filterEntry.SetToolTip(lang.X(
				"tree.filter.toolTip.tag",
				"Filter by a set of tags. Seperate tags with spaces to use multiple.\n"+
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
	displayByRadio.SetSelected(byResouceOption)

	displayByContainer := layouts.NewRightExpandHBox(
		displayByLbl, displayByRadio,
	)

	leftSplit := layouts.NewTopExpandVBox(
		layouts.NewBottomExpandVBox(
			displayByContainer,
			layouts.NewRightExpandHBox(
				widget.NewLabel(lang.X("tree.label", "Resources")),
				filterEntry,
			),
			container.NewStack(
				widgets.NewThemedRect(theme.ColorNameInputBackground, 4),
				container.NewPadded(tree),
			),
		),
	)

	defaultPreview := container.NewCenter(
		widget.NewLabel(lang.X("preview.defaultText", "Select a resource")),
	)

	rightSplit := container.NewStack(defaultPreview)

	bindings.Listen(treeSelected, func(tni string) {
		content := func() fyne.CanvasObject {
			if strings.HasPrefix(tni, "res://") {
				info := a.pkg.FileList().Find(func(fi *structures.FileInfo) bool {
					return fi.ResPath == tni
				})
				if info == nil {
					return defaultPreview
				}
				return a.buildInfoPane(info, editable)
			} else {
				_ = resourcesUnderTreeUID(tni, nodeTree)
				// else if strings.HasPrefix(tni, "tag://") {
				// tag := strings.TrimPrefix(tni, "tag://")
				// TODO: add bulk operations pane
			}
			return defaultPreview
		}()

		rightSplit.RemoveAll()
		rightSplit.Add(content)
		rightSplit.Refresh()
	})

	split := container.NewPadded(container.NewHSplit(
		container.NewPadded(leftSplit),
		container.NewPadded(rightSplit),
	))

	return split
}

func (a *App) buildPackageTree(editable bool) (*widget.Tree, binding.String, binding.String, binding.Bool, map[string][]string) {
	filterFunc := func(fi *structures.FileInfo) bool {
		return !fi.IsThumbnail() && !strings.HasSuffix(fi.ResPath, ".json")
	}
	nodeTree := make(map[string][]string)

	filter := ""
	boundFilter := binding.BindString(&filter)
	selected := binding.NewString()
	byTag := false
	boundByTag := binding.BindBool(&byTag)

	var tree *widget.Tree
	tree = widget.NewTree(
		func(tni widget.TreeNodeID) []widget.TreeNodeID {
			nodes, ok := nodeTree[tni]
			if ok {
				return nodes
			} else {
				return []string{}
			}
		},
		func(tni widget.TreeNodeID) bool {
			return !strings.HasPrefix(tni, "res://") && !strings.HasPrefix(tni, "empty://")
		},
		func(b bool) fyne.CanvasObject {
			var icon, btn fyne.CanvasObject
			if b {
				icon = widget.NewIcon(nil)
				btn = container.NewPadded(widgets.NewToolTipButtonWithIcon("template", theme.ErrorIcon(), nil))
			} else {
				icon = widget.NewFileIcon(nil)
				btn = nil
			}
			return container.NewBorder(nil, nil, icon, btn, widget.NewLabel("label template"))
		},
		func(tni widget.TreeNodeID, b bool, obj fyne.CanvasObject) {
			c := obj.(*fyne.Container)

			l := c.Objects[0].(*widget.Label)
			file := filepath.Base(tni)
			if b {
				icn := c.Objects[1].(*widget.Icon)
				btn := c.Objects[2].(*fyne.Container).Objects[0].(*widgets.ToolTipButton)
				var r fyne.Resource
				if tree.IsBranchOpen(tni) {
					r = theme.FolderOpenIcon()
				} else {
					r = theme.FolderIcon()
				}
				icn.SetResource(r)
				l.SetText(file)
				if editable && strings.HasPrefix(tni, "tag://") {
					btn.SetIcon(theme.DeleteIcon())
					tag := strings.TrimPrefix(tni, "tag://")
					btn.OnTapped = func() {
						dialog.ShowConfirm(
							lang.X("package.tag.delete.title", "Confirm Delete Tag"),
							lang.X(
								"package.tag.delete.message",
								"Do you want to delete the '{{.Tag}}' tag?",
								map[string]string{
									"Tag": tag,
								},
							),
							func(confirmed bool) {
								if confirmed {
									a.pkg.Tags().DeleteTag(tag)
									a.pkg.SaveUnpackedTags()
								}
							},
							a.window,
						)
					}
					btn.SetText("")
					btn.SetToolTip(lang.X(
						"package.tag.delete.tooltip",
						"Delete the '{{.Tag}}' tag",
						map[string]string{
							"Tag": tag,
						},
					))
					btn.Show()
				} else {
					btn.Hide()
				}
			} else {
				icn := c.Objects[1].(*widget.FileIcon)
				if strings.HasPrefix(tni, "empty://") {
					l.TextStyle = fyne.TextStyle{Italic: true}
					l.SetText(lang.X("tree.empty", "No resources"))
					icn.SetURI(nil)
				} else {
					icn.SetURI(
						storage.NewFileURI(
							filepath.Join(a.pkg.UnpackedPath(), utils.NormalizeResourcePath(tni)),
						))
					l.TextStyle = fyne.TextStyle{}
					l.SetText(file)
				}
			}
		},
	)

	filteredList := func() (*structures.FileInfoList, error) {
		filtered := a.pkg.FileList().Filter(filterFunc)
		if filter == "" {
			return filtered, nil
		}
		if byTag {
			tagFilter := ParseTagFilter(filter)
			log.Tracef("filtering tree list with %#v", tagFilter)
			return filtered.Filter(func(fi *structures.FileInfo) bool {
				ret := tagFilter.Apply(a.pkg.Tags().TagsFor(fi.ResPath))
				log.Tracef("%s matches = %t", fi.ResPath, ret)
				return ret
			}), nil
		}
		log.Tracef("filtering tree list with '%s'", filter)
		return filtered.Glob(nil, filter)
	}

	rebuildTree := func() {
		fil, err := filteredList()
		if err != nil {
			log.WithError(err).Error("failed to retrieve filtered file list")
			return
		}
		log.Trace("rebuilding tree")
		if byTag {
			nodeTree = buildTagMaps(fil, a.pkg.Tags(), filter)
		} else {
			nodeTree = buildInfoMaps(fil)
		}
		tree.Refresh()
		tree.OpenBranch(packageTreeRootID())
	}

	bindings.AddListenerToAll(
		rebuildTree,
		boundFilter,
		boundByTag,
		a.packageUpdated,
	)

	tree.OnSelected = func(uid widget.TreeNodeID) {
		selected.Set(uid)
	}

	return tree, boundFilter, selected, boundByTag, nodeTree
}

func (a *App) buildInfoPane(info *structures.FileInfo, editable bool) fyne.CanvasObject {
	tabs := make(map[string]*container.TabItem, 3)
	tabs["Resource"] = container.NewTabItemWithIcon(
		lang.X("preview.tab.resource", "Resource"),
		theme.FileIcon(),
		a.buildFilePreview(info),
	)

	tabsContainer := container.NewAppTabs(tabs["Resource"])

	if info.IsTaggable() {
		tabs["Tags"] = container.NewTabItemWithIcon(
			lang.X("preview.tab.tags", "Tags"),
			theme.ListIcon(),
			a.buildTagInfo(info, editable),
		)
		tabsContainer.Append(tabs["Tags"])
	}

	if info.ShouldHaveMetadata() {
		tabs["Settings"] = container.NewTabItemWithIcon(
			lang.X("preview.tab.metadata", "Settings"),
			theme.ColorPaletteIcon(),
			a.buildMetadataPane(info, editable),
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
		tabsContainer,
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
	thumbToggle := layouts.NewLeftExpandHBox(thumbnailLbl, thumbnailToggle)
	if !info.IsTexture() {
		thumbToggle.Hide()
	}

	pathText := widget.NewEntry()
	pathText.Disable()
	pathText.SetText(info.ResPath)
	path := container.NewStack(
		widgets.NewThemedRect(theme.ColorNameHeaderBackground, 4),
		layouts.NewRightExpandHBox(
			container.NewCenter(
				widget.NewLabel(lang.X(
					"preview.path.label",
					"Path",
				)),
			),
			container.NewPadded(
				container.NewStack(
					widgets.NewThemedRect(theme.ColorNameInputBackground, 4),
					container.NewPadded(
						container.NewHScroll(
							pathText,
						),
					),
				),
			),
		),
	)

	resMd5Text := widget.NewEntry()
	resMd5Text.Disable()

	resMd5 := container.NewStack(
		widgets.NewThemedRect(theme.ColorNameHeaderBackground, 4),
		layouts.NewRightExpandHBox(
			container.NewCenter(
				widget.NewLabel(lang.X(
					"preview.md5.label",
					"Md5 Hash",
				)),
			),
			container.NewPadded(
				container.NewStack(
					widgets.NewThemedRect(theme.ColorNameInputBackground, 4),
					container.NewPadded(
						resMd5Text,
					),
				),
			),
		),
	)

	a.pkg.GetOrUpdateResourceMd5(info, func(md5 string, err error) {
		if err != nil {
			log.WithError(err).Errorf("Failed to update hash for %s", info.ResPath)
		}
		resMd5Text.SetText(md5)
	})

	tooLarge := container.NewCenter(
		widget.NewLabel(lang.X("preview.tooLarge", "This file is too large!\nOpen it in a text editor.")),
	)

	bg := widgets.NewThemedRect(theme.ColorNameInputBackground, 4)

	if !ddimage.PathIsSupportedImage(info.RelPath) {
		textContent := string(fileData)
		if len(strings.Split(textContent, "\n")) > 200 {
			return container.NewPadded(layouts.NewBottomExpandVBox(path, resMd5, container.NewStack(
				bg, tooLarge,
			)))
		}
		widget.NewMultiLineEntry()
		textEntry := widget.NewMultiLineEntry()
		textEntry.Text = textContent
		copyBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
			a.app.Clipboard().SetContent(string(fileData))
		})
		content := container.NewPadded(
			layouts.NewBottomExpandVBox(
				path,
				resMd5,
				container.NewStack(
					bg,
					container.NewPadded(textEntry),
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
	imgW := canvas.NewImageFromImage(img)

	imgW.FillMode = canvas.ImageFillOriginal
	imgW.ScaleMode = canvas.ImageScaleFastest

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
		path,
		resMd5,
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
					tagsList,
				),
			),
		),
	)

	if editable {
		tagSelecter := widget.NewSelectEntry(a.pkg.Tags().AllTags())
		addBtn := widget.NewButtonWithIcon(
			lang.X("tags.addBtn.text", "Add"),
			theme.ContentAddIcon(),
			func() {
				if tagSelecter.Text == "" {
					return
				}
				a.pkg.Tags().Tag(tagSelecter.Text, info.RelPath)
				updateTags()
				a.saveUnpackedTags()
				tagSelecter.SetText("")
				tagSelecter.SetOptions(a.pkg.Tags().AllTags())
			})
		content.Add(layouts.NewLeftExpandHBox(tagSelecter, addBtn))
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
			form := container.New(
				layout.NewFormLayout(),
				colorLbl, colorRect,
			)
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

			form := container.New(
				layout.NewFormLayout(),
				nameLbl, nameEntry,
				tilesetTypeLbl, tilesetTypeSelector,
				colorLbl, colorRect,
			)
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

func nodeID(dir string) string {
	id := "dir://" + dir
	if dir == "" {
		id = packageTreeRootID()
	}
	return id
}

func packageTreeRootID() string {
	return "root://" + lang.X("tree.root.label", "Package")
}

func resourcesUnderTreeUID(tni string, tree map[string][]string) []string {
	res := []string{}
	stack := structures.NewStack[string]()
	stack.Push(tni)
	for !stack.IsEmpty() {
		uid, _ := stack.Pop()
		leaves, exists := tree[uid]
		if exists {
			for _, leaf := range leaves {
				if strings.HasPrefix(leaf, "res://") {
					res = append(res, uid)
				} else {
					stack.Push(uid)
				}
			}
		}
	}
	return res
}

func buildInfoMaps(fil *structures.FileInfoList) map[string][]string {
	nodeTree := make(map[string][]string)

	for _, fi := range fil.AsSlice() {
		dir, _ := filepath.Split(fi.RelPath)
		next := dir[:max(len(dir)-1, 0)]
		node := nodeID(next)
		path := fi.RelPath
		var nodeLeaf string
		if !slices.Contains(nodeTree[node], fi.ResPath) {
			nodeTree[node] = append(nodeTree[node], fi.ResPath)
		}
		for next != "" {
			path = next
			dir, _ = filepath.Split(next)
			next = dir[:max(len(dir)-1, 0)]
			node = nodeID(next)
			nodeLeaf = "dir://" + path
			if !slices.Contains(nodeTree[node], nodeLeaf) {
				nodeTree[node] = append(nodeTree[node], nodeLeaf)
			}
		}
	}

	rootID := packageTreeRootID()

	if len(nodeTree[rootID]) == 0 {
		nodeTree[binding.DataTreeRootID] = append(nodeTree[binding.DataTreeRootID], "empty://")
	} else {
		nodeTree[binding.DataTreeRootID] = append(nodeTree[binding.DataTreeRootID], rootID)
	}

	return nodeTree
}

func buildTagMaps(fil *structures.FileInfoList, pt *structures.PackageTags, filter string) map[string][]string {
	nodeTree := make(map[string][]string)
	untaggedPath := "notag://" + lang.X("tree.untagged.label", "Untagged")
	for _, fi := range fil.AsSlice() {
		if fi.IsTaggable() {
			tags := pt.TagsFor(fi.ResPath)
			if tags.Size() == 0 {
				nodeTree[untaggedPath] = append(nodeTree[untaggedPath], fi.ResPath)
			} else {
				for _, tag := range tags.AsSlice() {
					nodeTree["tag://"+tag] = append(nodeTree["tag://"+tag], fi.ResPath)
				}
			}
		}
	}

	allTags := pt.AllTags()
	slices.Sort(allTags)
	tagFilter := ParseTagFilter(filter)

	rootID := packageTreeRootID()

	for _, tag := range allTags {
		if filter != "" && !tagFilter.ApplyToTag(tag) {
			continue
		}
		if len(nodeTree["tag://"+tag]) == 0 {
			nodeTree["tag://"+tag] = append(nodeTree["tag://"+tag], "empty://"+tag)
		}
		nodeTree[rootID] = append(nodeTree[rootID], "tag://"+tag)
	}

	if len(nodeTree[untaggedPath]) > 0 {
		nodeTree[rootID] = append(nodeTree[rootID], untaggedPath)
	}

	if len(nodeTree[rootID]) == 0 {
		nodeTree[binding.DataTreeRootID] = append(nodeTree[binding.DataTreeRootID], "empty://")
	} else {
		nodeTree[binding.DataTreeRootID] = append(nodeTree[binding.DataTreeRootID], rootID)
	}

	return nodeTree
}
