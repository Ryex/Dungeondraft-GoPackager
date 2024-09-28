package gui

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/davecgh/go-spew/spew"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/bindings"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/layouts"
	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddimage"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
	log "github.com/sirupsen/logrus"
)

func (a *App) buildPackageTreeAndInfoPane(editable bool) fyne.CanvasObject {
	filter := binding.NewString()
	filterEntry := widget.NewEntryWithData(filter)
	filterEntry.SetPlaceHolder(lang.X("tree.filter.placeholder", "Filter with glob (ie. */objects/**)"))
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

	tree, treeSelected := a.buildPackageTree(filter)

	leftSplit := container.New(
		layouts.NewBottomExpandVBoxLayout(),
		container.New(
			layouts.NewRightExpandHBoxLayout(),
			widget.NewLabel(lang.X("tree.label", "Resources")),
			filterEntry,
		),
		container.NewStack(
			canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground)),
			tree,
		),
	)

	defaultPreview := container.NewCenter(
		widget.NewLabel(lang.X("preview.defaultText", "Select a resource")),
	)

	rightSplit := container.NewStack(defaultPreview)

	bindings.Listen(treeSelected, func(tni string) {
		content := func() fyne.CanvasObject {
			info := a.pkg.FileList().Find(func(fi *structures.FileInfo) bool {
				return fi.RelPath == tni
			})
			if info == nil {
				return defaultPreview
			}
			return a.buildInfoPane(info, editable)
		}()

		rightSplit.RemoveAll()
		rightSplit.Add(content)
		rightSplit.Refresh()
	})

	split := container.NewPadded(container.NewHSplit(
		leftSplit,
		container.NewPadded(rightSplit),
	))

	return split
}

func (a *App) buildPackageTree(filter binding.String) (*widget.Tree, binding.String) {
	filterFunc := func(fi *structures.FileInfo) bool {
		return !fi.IsThumbnail() && !strings.HasSuffix(fi.ResPath, ".json")
	}
	mappedList := bindings.NewMapping(
		filter,
		func(filter string) ([]structures.FileInfo, error) {
			log.Tracef("filtering tree list with '%s'", filter)
			if filter == "" {
				return a.pkg.FileList().Filter(filterFunc), nil
			}
			return a.pkg.FileList().Glob(filterFunc, filter)
		},
	)
	nodeTree := make(map[string][]string)

	selected := binding.NewString()

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
			if a.pkg == nil {
				return false
			}
			info := a.pkg.FileList().Find(func(fi *structures.FileInfo) bool {
				return fi.RelPath == tni
			})
			return info == nil
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
			_, file := filepath.Split(tni)
			if b {
				var r fyne.Resource
				if tree.IsBranchOpen(tni) {
					r = theme.FolderOpenIcon()
				} else {
					r = theme.FolderIcon()
				}
				c.Objects[1].(*widget.Icon).SetResource(r)
				l.SetText(file + "/")
			} else {
				c.Objects[1].(*widget.FileIcon).SetURI(storage.NewFileURI(tni))
				l.SetText(file)
			}
		},
	)

	bindings.ListenErr(mappedList, func(fil []structures.FileInfo) {
		log.Trace("rebuilding tree")
		nodeTree = buildInfoMaps(fil)
		tree.Refresh()
	}, func(err error) {
		log.WithError(err).Debug("file list fetch failure")
	})

	tree.OnSelected = func(uid widget.TreeNodeID) {
		selected.Set(uid)
	}

	return tree, selected
}

func (a *App) buildInfoPane(info *structures.FileInfo, editable bool) fyne.CanvasObject {
	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon(
			lang.X("preview.tab.resource", "Resource"),
			theme.FileIcon(),
			a.buildFilePreview(info)),
	)

	if info.IsTexture() {
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
			nil,
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

	pathLabel := widget.NewLabel(
		lang.X(
			"preview.path.label",
			"Path",
		),
	)
	pathEntry := widget.NewEntry()
	pathEntry.SetText(info.CalcRelPath())
	pathEntry.OnChanged = func(_ string) {
		pathEntry.SetText(info.CalcRelPath())
	}
	pathEntry.Refresh()
	path := container.New(
		layouts.NewRightExpandHBoxLayout(),
		pathLabel,
		pathEntry,
	)

	tooLarge := container.NewCenter(
		widget.NewLabel(lang.X("preview.toolarge", "This file is too large!\nOpen it in a text editor.")),
	)

	bg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	if !ddimage.PathIsSupportedImage(info.RelPath) {
		textContent := string(fileData)
		if len(textContent) > 1000 {
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
					textEntry,
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
			container.NewScroll(
				imgW,
			),
		)),
	))
	return content
}

func (a *App) buildTagInfo(info *structures.FileInfo, editable bool) fyne.CanvasObject {
	var tags []string
	boundTags := binding.BindStringList(&tags)
	updateTags := func() {
		updated := a.pkg.Tags().TagsFor(info.RelPath).AsSlice()
		boundTags.Set(updated)
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
	content := layouts.NewTopExpandVBox(tagsList)

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
			})
		content.Add(layouts.NewLeftExpandHBox(tagSelecter, addBtn))
	}

	return content
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

func buildInfoMaps(infoList []structures.FileInfo) map[string][]string {
	nodeTree := make(map[string][]string)
	for i := 0; i < len(infoList); i++ {
		info := &(infoList)[i]

		dir, _ := filepath.Split(info.RelPath)
		next := dir[:max(len(dir)-1, 0)]
		path := info.RelPath
		nodeTree[next] = append(nodeTree[next], path)
		for next != "" {
			path = next
			dir, _ = filepath.Split(next)
			next = dir[:max(len(dir)-1, 0)]
			if !utils.InSlice(path, nodeTree[next]) {
				nodeTree[next] = append(nodeTree[next], path)
			}
		}
		if !utils.InSlice(path, nodeTree[next]) {
			nodeTree[next] = append(nodeTree[next], path)
		}
	}

	return nodeTree
}
