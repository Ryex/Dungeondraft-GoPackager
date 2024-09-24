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

	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddimage"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
	log "github.com/sirupsen/logrus"
)

func (a *App) loadUnpackedPath(path string) {
	activity := a.setWaitContent(lang.X("pack.wait", "Loading unpacked resources from {{.Path}} ...", map[string]any{"Path": path}))
	a.disableButtons.Set(true)

	if a.pkgFile != nil {
		a.pkgFile.Close()
		a.pkgFile = nil
	}
	a.pkg = nil

	go func() {
		l := log.WithFields(log.Fields{
			"path": path,
		})

		pkg := ddpackage.NewPackage(l)

		err := pkg.LoadUnpackedFromFolder(path)
		if err != nil {
			l.WithError(err).Error("could not load directory")
			a.setErrContent(
				err,
				lang.X("err.badResources", "Failed to load unpacked resources from {{.Path}}", map[string]any{"Path": path}),
			)
			return
		}

		err = pkg.BuildFileList(func(path string) {
			activity.Set(lang.X("pack.buildList.activity", "Loading {{.Path}} ...", map[string]any{"Path": path}))
		})
		if err != nil {
			l.WithError(err).Error("could not build file list")
			a.setErrContent(
				err,
				lang.X("err.fileList", "Failed to build file list from {{.Path}}", map[string]any{"Path": path}),
			)
			return
		}
		a.pkgFile = nil
		a.setUnpackedContent(pkg)
	}()
}

func (a *App) setUnpackedContent(pkg *ddpackage.Package) {
	a.pkg = pkg

	tree, treeSelected, nodeMap := a.buildPackageTree()

	refreshBtn := widget.NewButtonWithIcon(lang.X("pack.refreshBtn.text", "Refresh"), theme.ViewRefreshIcon(), func() {
		a.loadUnpackedPath(a.pkg.UnpackedPath)
	})

	leftSplit := container.New(
		NewBottomExpandVBoxLayout(),
		container.New(
			NewLeftExpandHBoxLayout(),
			widget.NewLabel(lang.X("pack.tree.label", "Resources")),
			refreshBtn,
		),
		container.NewStack(
			canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground)),
			tree,
		),
	)

	defaultPreview := container.NewCenter(
		widget.NewLabel(lang.X("pack.preview.defaultText", "Select a resource")),
	)

	rightSplit := container.NewStack(defaultPreview)

	treeSelected.AddListener(binding.NewDataListener(func() {
		tni, err := treeSelected.Get()
		if err != nil {
			log.WithError(err).Error("error collecting bound tree node value")
			return
		}

		content := func() fyne.CanvasObject {
			info, ok := nodeMap[tni]
			if !ok {
				return defaultPreview
			}

			filePath := info.Path

			fileData, err := utils.ReadFile(info.Path)
			if err != nil {
				log.WithError(err).Errorf("failed to read image data for %s", filePath)
				return widget.NewLabel(fmt.Sprintf("Failed to read image data for %s", filePath))
			}

			label := widget.NewLabel(lang.X(
				"pack.preview.label",
				"Path: {{.Path}}",
				map[string]any{
					"Path": filepath.Clean(info.RelPath),
				},
			))

			if !ddimage.PathIsSupportedImage(info.RelPath) {
				widget.NewMultiLineEntry()
				textGrid := widget.NewTextGridFromString(string(fileData))
				textGrid.ShowLineNumbers = true
				bg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
				copyBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
					a.window.Clipboard().SetContent(string(fileData))
				})
				content := container.NewPadded(
					container.New(
						NewBottomExpandVBoxLayout(),
						label,
						container.NewStack(
							bg,
							textGrid,
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

			img, err := ddimage.BytesToImage(fileData)
			if err != nil {
				log.WithError(err).Errorf("failed to decode image for %s", filePath)
				content := widget.NewLabel(fmt.Sprintf("Failed to decode image for %s", filePath))
				return content
			}

			log.Infof("loaded image for %s", filePath)
			imgW := canvas.NewImageFromImage(img)
			height := float32(64)
			if strings.HasPrefix(tni, "textures/terrain") {
				height = 160
			} else if strings.HasPrefix(tni, "textures/walls") {
				height = 32
			} else if strings.HasPrefix(tni, "textures/paths") {
				height = 48
			}
			tmpW := float64(height) * float64(img.Bounds().Dx()) / float64(img.Bounds().Dy())
			width := float32(math.Max(1.0, math.Floor(tmpW+0.5)))
			imgW.SetMinSize(fyne.NewSize(width, height))
			imgW.FillMode = canvas.ImageFillContain
			imgW.ScaleMode = canvas.ImageScaleFastest
			content := container.New(
				NewBottomExpandVBoxLayout(),
				label,
				imgW,
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

	disableButtonsListener := binding.NewDataListener(func() {
		disable, _ := a.disableButtons.Get()
		if disable {
		} else {
		}
	})

	a.setMainContent(
		container.New(
			NewBottomExpandVBoxLayout(),
			split,
		),
		disableButtonsListener,
	)

	a.disableButtons.Set(false)
}
