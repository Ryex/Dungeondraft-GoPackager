package gui

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/custom_binding"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/custom_layout"
	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddimage"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
	log "github.com/sirupsen/logrus"
)

func (a *App) buildPackageTreeAndPreview() fyne.CanvasObject {
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
		custom_layout.NewBottomExpandVBoxLayout(),
		container.New(
			custom_layout.NewRightExpandHBoxLayout(),
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
	tooLarge := container.NewCenter(
		widget.NewLabel(lang.X("preview.toolarge", "This file is too large!\nOpen it in a text editor.")),
	)

	rightSplit := container.NewStack(defaultPreview)

	treeSelected.AddListener(binding.NewDataListener(func() {
		tni, err := treeSelected.Get()
		if err != nil {
			log.WithError(err).Error("error collecting bound tree node value")
			return
		}

		content := func() fyne.CanvasObject {
			info := a.pkg.FileList().Find(func(fi *structures.FileInfo) bool {
				return fi.RelPath == tni
			})
			if info == nil {
				return defaultPreview
			}

			fileData, err := a.pkg.LoadResource(info.ResPath)
			if err != nil {
				log.WithError(err).Errorf("failed to read image data for %s", tni)
				return widget.NewLabel(fmt.Sprintf("Failed to read image data for %s", tni))
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
				custom_layout.NewRightExpandHBoxLayout(),
				pathLabel,
				pathEntry,
			)

			if !ddimage.PathIsSupportedImage(info.RelPath) {
				textContent := string(fileData)
				if len(textContent) > 1000 {
					return tooLarge
				}
				widget.NewMultiLineEntry()
				textEntry := widget.NewMultiLineEntry()
				textEntry.Text = textContent
				bg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
				copyBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
					a.window.Clipboard().SetContent(string(fileData))
				})
				content := container.NewPadded(
					container.New(
						custom_layout.NewBottomExpandVBoxLayout(),
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
				log.WithError(err).Errorf("failed to decode image for %s", tni)
				content := widget.NewLabel(fmt.Sprintf("Failed to decode image for %s", tni))
				return content
			}

			log.Infof("loaded image for %s", tni)
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
			content := container.New(
				custom_layout.NewBottomExpandVBoxLayout(),
				path,
				container.NewScroll(
					imgW,
				),
			)
			return content
		}()

		rightSplit.RemoveAll()
		rightSplit.Add(content)
		rightSplit.Refresh()
	}))

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
	mappedList := custom_binding.NewMapping(
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

	tree := widget.NewTree(
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
			return widget.NewLabel("label template")
		},
		func(tni widget.TreeNodeID, b bool, obj fyne.CanvasObject) {
			l := obj.(*widget.Label)
			_, file := filepath.Split(tni)
			if b {
				l.SetText(file + "/")
			} else {
				l.SetText(file)
			}
		},
	)

	filter.AddListener(binding.NewDataListener(func() {
		log.Trace("rebuilding tree")
		fil, err := mappedList.Get()
		if err != nil {
			log.WithError(err).Debug("file list fetch failure")
			return
		}
		nodeTree = buildInfoMaps(fil)
		tree.Refresh()
	}))

	tree.OnSelected = func(uid widget.TreeNodeID) {
		selected.Set(uid)
	}

	return tree, selected
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
