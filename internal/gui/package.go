package gui

import (
	"fmt"
	"image/color"
	"math"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/davecgh/go-spew/spew"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/bindings"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/layouts"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/widgets"
	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddimage"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
	ddcolor "github.com/ryex/dungeondraft-gopackager/pkg/structures/color"
	log "github.com/sirupsen/logrus"
)

func (a *App) buildPackageTreeAndInfoPane(editable bool) fyne.CanvasObject {
	tree, filter, treeSelected, displayByTag := a.buildPackageTree()

	filterEntry := widget.NewEntryWithData(filter)
	filterEntry.Validator = nil
	filterEntry.SetPlaceHolder(lang.X("tree.filter.placeholder.resource", "Filter with glob (ie. */objects/**)"))
	filterErrorText := canvas.NewText(lang.X("tree.filter.error", "Bad glob syntax"), theme.Color(theme.ColorNameError))
	filterErrorText.Hide()
	filterEntry.Validator = func(s string) error {
		_, err := structures.GlobToRelPathRegexp(s)
		if err != nil {
			filterErrorText.Show()
		} else {
			filterErrorText.Hide()
		}
		return err
	}

	displayByLbl := widget.NewLabel(lang.X("tree.displayBy.label", "Display By"))

	byResouceOption := lang.X("tree.displayby.resource", "Resource")
	byTagOption := lang.X("tree.displayby.tag", "Tag")

	displayByRadio := widget.NewRadioGroup([]string{byResouceOption, byTagOption}, func(selected string) {
		if selected == byResouceOption {
			displayByTag.Set(false)
			filterEntry.SetPlaceHolder(lang.X("tree.filter.placeholder.resource", "Filter with glob (ie. */objects/**)"))
		} else {
			displayByTag.Set(true)
			filterEntry.SetPlaceHolder(lang.X("tree.filter.placeholder.tags", "Filter by tag name"))
		}
	})
	displayByRadio.Required = true
	displayByRadio.Horizontal = true
	displayByRadio.SetSelected(byResouceOption)

	displayByContainer := layouts.NewRightExpandHBox(
		displayByLbl, displayByRadio,
	)
	tagSetsBtn := widget.NewButton(
		lang.X("tagSetsBtn.text", "Edit Tag Sets"),
		func() {
			dlg := a.createTagSetsDialog(editable)
			dlg.Show()
		},
	)

	leftSplit := layouts.NewTopExpandVBox(
		layouts.NewBottomExpandVBox(
			displayByContainer,
			layouts.NewRightExpandHBox(
				widget.NewLabel(lang.X("tree.label", "Resources")),
				filterEntry,
			),
			container.NewStack(
				&canvas.Rectangle{
					FillColor: theme.Color(theme.ColorNameInputBackground),
				},
				container.NewPadded(tree),
			),
		),
		tagSetsBtn,
	)

	if editable {
		generateTageBtn := widget.NewButton(
			lang.X("generateTageBtn.label", "Generate Tags"),
			func() {
				dlg := a.createTagGenDialog()
				dlg.Show()
			})
		leftSplit.Add(generateTageBtn)
	}

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

func (a *App) buildPackageTree() (*widget.Tree, binding.String, binding.String, binding.Bool) {
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
			return !strings.HasPrefix(tni, "res://")
		},
		func(b bool) fyne.CanvasObject {
			var icon fyne.CanvasObject
			if b {
				icon = widget.NewIcon(nil)
			} else {
				icon = widget.NewFileIcon(nil)
			}
			return container.NewBorder(nil, nil, icon, nil, widget.NewLabel("label template"))
		},
		func(tni widget.TreeNodeID, b bool, obj fyne.CanvasObject) {
			c := obj.(*fyne.Container)

			l := c.Objects[0].(*widget.Label)
			file := filepath.Base(tni)
			if b {
				var r fyne.Resource
				if tree.IsBranchOpen(tni) {
					r = theme.FolderOpenIcon()
				} else {
					r = theme.FolderIcon()
				}
				c.Objects[1].(*widget.Icon).SetResource(r)
				l.SetText(file)
			} else {
				c.Objects[1].(*widget.FileIcon).SetURI(
					storage.NewFileURI(
						filepath.Join(a.pkg.UnpackedPath(), utils.NormalizeResourcePath(tni)),
					))
				l.SetText(file)
			}
		},
	)

	tree.OnBranchOpened = func(uid widget.TreeNodeID) {
		log.Infof("node %s opened| children %v", uid, nodeTree[uid])
	}

	filteredList := func() ([]*structures.FileInfo, error) {
		filtered := a.pkg.FileList().Filter(filterFunc)
		if filter == "" {
			return filtered, nil
		}
		if byTag {
			return filtered.Filter(func(fi *structures.FileInfo) bool {
				return utils.Any(a.pkg.Tags().TagsFor(fi.ResPath).AsSlice(), func(tag string) bool {
					return strings.Contains(strings.ToLower(tag), strings.ToLower(filter))
				})
			}), nil
		}
		log.Tracef("filtering tree list with '%s'", filter)
		return filtered.Glob(nil, filter)
	}

	rebuildTree := func() {
		fil, err := filteredList()
		if err != nil {
			log.WithError(err).Error("failed to retrieve filtered file list")
		}
		log.Trace("rebuilding tree")
		if byTag {
			nodeTree = buildTagMaps(fil, a.pkg.Tags())
		} else {
			nodeTree = buildInfoMaps(fil)
		}
		tree.Refresh()
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

	return tree, boundFilter, selected, boundByTag
}

func (a *App) buildInfoPane(info *structures.FileInfo, editable bool) fyne.CanvasObject {
	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon(
			lang.X("preview.tab.resource", "Resource"),
			theme.FileIcon(),
			a.buildFilePreview(info)),
	)

	if info.IsTaggable() {
		tabs.Append(container.NewTabItemWithIcon(
			lang.X("preview.tab.tags", "Tags"),
			theme.ListIcon(),
			a.buildTagInfo(info, editable),
		))
	}

	if info.ShouldHaveMetadata() {
		tabs.Append(container.NewTabItemWithIcon(
			lang.X("preview.tab.metadata", "Settings"),
			theme.ColorPaletteIcon(),
			a.buildMetadataPane(info, editable),
		))
	}

	tabs.SetTabLocation(container.TabLocationTop)
	return tabs
}

func (a *App) buildFilePreview(info *structures.FileInfo) fyne.CanvasObject {
	fileData, err := a.pkg.LoadResource(info.ResPath)
	if err != nil {
		log.WithError(err).Errorf("failed to read image data for %s", info.ResPath)
		return widget.NewLabel(fmt.Sprintf("Failed to read image data for %s", info.ResPath))
	}

	path := container.NewStack(
		&canvas.Rectangle{
			FillColor:    theme.Color(theme.ColorNameHeaderBackground),
			CornerRadius: 4,
		},
		layouts.NewRightExpandHBox(
			container.NewCenter(
				widget.NewLabel(lang.X(
					"preview.path.label",
					"Path",
				)),
			),
			container.NewPadded(
				container.NewStack(
					&canvas.Rectangle{
						FillColor:    theme.Color(theme.ColorNameInputBackground),
						CornerRadius: 4,
					},
					container.NewPadded(
						container.NewHScroll(
							canvas.NewText(info.ResPath, theme.Color(theme.ColorNameForeground)),
						),
					),
				),
			),
		),
	)

	tooLarge := container.NewCenter(
		widget.NewLabel(lang.X("preview.tooLarge", "This file is too large!\nOpen it in a text editor.")),
	)

	bg := &canvas.Rectangle{
		FillColor:    theme.Color(theme.ColorNameInputBackground),
		CornerRadius: 4,
	}
	if !ddimage.PathIsSupportedImage(info.RelPath) {
		textContent := string(fileData)
		if len(strings.Split(textContent, "\n")) > 200 {
			return container.NewPadded(layouts.NewBottomExpandVBox(path, container.NewStack(
				bg, tooLarge,
			)))
		}
		widget.NewMultiLineEntry()
		textEntry := widget.NewMultiLineEntry()
		textEntry.Text = textContent
		copyBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
			a.window.Clipboard().SetContent(string(fileData))
		})
		content := container.NewPadded(
			layouts.NewBottomExpandVBox(
				path,
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
		content := widget.NewLabel(fmt.Sprintf("Failed to decode image for %s", info.ResPath))
		return content
	}

	log.Infof("loaded image for %s", info.ResPath)
	imgW := canvas.NewImageFromImage(img)
	height := float32(64)
	if info.IsTerrain() {
		height = 160
	} else if info.IsWall() {
		height = 32
	} else if info.IsPath() {
		height = 48
	}
	tmpW := float64(height) * float64(img.Bounds().Dx()) / float64(img.Bounds().Dy())
	width := float32(math.Max(1.0, math.Floor(tmpW+0.5)))
	imgW.SetMinSize(fyne.NewSize(width, height))
	imgW.FillMode = canvas.ImageFillContain
	imgW.ScaleMode = canvas.ImageScaleFastest
	content := container.NewPadded(container.New(
		layouts.NewBottomExpandVBoxLayout(),
		path,
		container.NewPadded(container.NewStack(
			bg,
			container.New(
				layout.NewCustomPaddedLayout(8, 8, 8, 8),
				container.NewScroll(
					imgW,
				),
			),
		)),
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
				widget.NewButtonWithIcon("", theme.DeleteIcon(), nil),
			)
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			c := co.(*fyne.Container)
			l := c.Objects[0].(*widget.Label)
			l.Bind(di.(binding.String))
			btn := c.Objects[1].(*widget.Button)
			if editable {
				btn.OnTapped = func() {
					tag, err := di.(binding.String).Get()
					if err != nil {
						log.WithError(err).Errorf("failed to get tag in del tag btn for %s", info.RelPath)
						return
					}
					a.pkg.Tags().Untag(tag, info.RelPath)
					updateTags()
					a.saveUnpackedTags()
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
				&canvas.Rectangle{
					FillColor:    theme.Color(theme.ColorNameHeaderBackground),
					CornerRadius: 4,
				},
				layouts.NewRightExpandHBox(
					container.NewCenter(
						widget.NewLabel(lang.X(
							"preview.tab.tags.label",
							"Tags for",
						)),
					),
					container.NewPadded(
						container.NewStack(
							&canvas.Rectangle{
								FillColor:    theme.Color(theme.ColorNameInputBackground),
								CornerRadius: 4,
							},
							container.NewPadded(
								container.NewHScroll(
									canvas.NewText(info.ResPath, theme.Color(theme.ColorNameForeground)),
								),
							),
						),
					),
				),
			),
			tagsList,
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

			defaultColor := color.NRGBA{255, 0, 0, 255}
			wallData := a.pkg.Walls()
			if wallData != nil {
				metaData, ok := (*wallData)[metaPath]
				if ok {
					defaultColor = metaData.Color.ToColor()
				} else {
					log.WithField("res", info.ResPath).
						WithField("metaRes", metaPath).
						Warn("Missing wall metadata")
				}
			} else {
				log.Warn("Wall Metadata not loaded?")
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
								data, ok := (*wallData)[metaPath]
								if !ok {
									(*wallData)[metaPath] = structures.PackageWall{
										Path:  info.RelPath,
										Color: ddcolor.FromColor(c),
									}
								} else {
									data.Color = ddcolor.FromColor(c)
									(*wallData)[metaPath] = data
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

			defaultColor := color.NRGBA{255, 0, 0, 255}
			tilesetName := ""
			tilesetType := structures.TilesetNormal

			tilesetData := a.pkg.Tilesets()
			if tilesetData != nil {
				metaData, ok := (*tilesetData)[metaPath]
				if ok {
					defaultColor = metaData.Color.ToColor()
					tilesetName = metaData.Name
					tilesetType = metaData.Type
				} else {
					log.WithField("res", info.ResPath).
						WithField("metaRes", metaPath).
						Warn("Missing tileset metadata")
				}
			} else {
				log.Warn("Wall Metadata not loaded?")
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
								data, ok := (*tilesetData)[metaPath]
								if !ok {
									(*tilesetData)[metaPath] = structures.PackageTileset{
										Path:  info.RelPath,
										Name:  "",
										Color: ddcolor.FromColor(c),
										Type:  structures.TilesetNormal,
									}
								} else {
									data.Color = ddcolor.FromColor(c)
									(*tilesetData)[metaPath] = data
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
				data, ok := (*tilesetData)[metaPath]
				if !ok {
					(*tilesetData)[metaPath] = structures.PackageTileset{
						Path:  info.RelPath,
						Name:  s,
						Color: ddcolor.FromColor(defaultColor),
						Type:  structures.TilesetNormal,
					}
				} else {
					data.Name = s
					(*tilesetData)[metaPath] = data
				}
				a.saveTilesetMetadata(metaPath)
			}

			tilesetTypeLbl := widget.NewLabel(lang.X("metadata.tilesetType.label", "Tileset Type"))
			tilesetTypeSelector := widget.NewSelect(
				[]string{
					string(structures.TilesetNormal),
					string(structures.TilesetCustomColor),
				},
				func(s string) {
					var typ structures.TilesetType
					switch s {
					case string(structures.TilesetNormal):
						typ = structures.TilesetNormal
					case string(structures.TilesetCustomColor):
						typ = structures.TilesetCustomColor
					}
					data, ok := (*tilesetData)[metaPath]
					if !ok {
						(*tilesetData)[metaPath] = structures.PackageTileset{
							Path:  info.RelPath,
							Name:  "",
							Color: ddcolor.FromColor(defaultColor),
							Type:  typ,
						}
					} else {
						data.Type = typ
						(*tilesetData)[metaPath] = data
					}
					a.saveTilesetMetadata(metaPath)
				},
			)
			tilesetTypeSelector.SetSelected(string(tilesetType))

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
		id = binding.DataTreeRootID
	}
	return id
}

func buildInfoMaps(fil structures.FileInfoList) map[string][]string {
	nodeTree := make(map[string][]string)
	for _, fi := range fil {
		dir, _ := filepath.Split(fi.RelPath)
		next := dir[:max(len(dir)-1, 0)]
		node := nodeID(next)
		path := fi.RelPath
		var nodeLeaf string
		nodeTree[node] = append(nodeTree[node], fi.ResPath)
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

	return nodeTree
}

func buildTagMaps(fil structures.FileInfoList, pt *structures.PackageTags) map[string][]string {
	nodeTree := make(map[string][]string)
	for _, fi := range fil {
		if fi.IsTaggable() {
			tags := pt.TagsFor(fi.ResPath)
			if tags.Size() == 0 {
				nodeTree["notag://objects"] = append(nodeTree["notag://objects"], fi.ResPath)
			} else {
				for _, tag := range tags.AsSlice() {
					nodeTree["tag://"+tag] = append(nodeTree["tag://"+tag], fi.ResPath)
				}
			}
		}
	}
	nodeTree[binding.DataTreeRootID] = utils.Map(pt.AllTags(), func(tag string) string {
		return "tag://" + tag
	})
	return nodeTree
}
