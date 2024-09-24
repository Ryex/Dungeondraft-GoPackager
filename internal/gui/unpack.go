package gui

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
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

	"github.com/davecgh/go-spew/spew"
	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddimage"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"

	log "github.com/sirupsen/logrus"
)

func (a *App) loadPack(path string) {
	a.setWaitContent(lang.X("unpack.wait", "Loading {{.Path}} ...", map[string]any{"Path": path}))
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

		file, err := pkg.LoadFromPackedPath(path)
		if err != nil {
			l.WithError(err).Error("could not load path")
			a.setErrContent(
				err,
				lang.X("err.badPack", "Failed to load {{.Path}}", map[string]any{"Path": path}),
			)
			return
		}

		a.pkgFile = file
		a.setPackContent(pkg)
	}()
}

func (a *App) setPackContent(pkg *ddpackage.Package) {
	a.pkg = pkg

	tree, treeSelected, nodeMap := a.buildPackageTree()

	log.Info("tree built")

	leftSplit := container.New(
		NewBottomExpandVBoxLayout(),
		widget.NewLabel(lang.X("unpack.tree.label", "Resources")),
		container.NewStack(
			canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground)),
			tree,
		),
	)

	defaultPreview := container.NewCenter(
		widget.NewLabel(lang.X("unpack.preview.defaultText", "Select a resource")),
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

			fileData, err := a.pkg.ReadFileFromPackage(a.pkgFile, *info)
			if err != nil {
				log.WithError(err).Errorf("failed to read image data for %s", tni)
				return widget.NewLabel(fmt.Sprintf("Failed to read image data for %s", tni))
			}

			label := widget.NewLabel(lang.X("unpack.preview.label", "Path: {{.Path}}", map[string]any{"Path": info.ResPath}))

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
				log.WithError(err).Errorf("failed to decode image for %s", tni)
				content := widget.NewLabel(fmt.Sprintf("Failed to decode image for %s", tni))
				return content
			}

			log.Infof("loaded image for %s", tni)
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

	log.Info("tree slector built")
	split := container.NewPadded(container.NewHSplit(
		leftSplit,
		container.NewPadded(rightSplit),
	))

	outputPath := binding.NewString()

	outEntry := widget.NewEntryWithData(outputPath)
	outEntry.SetPlaceHolder(lang.X("unpack.outPath.placeholder", "Where to extract resources"))
	outBrowseBtn := widget.NewButtonWithIcon(lang.X("browse", "Browse"), theme.FileIcon(), func() {
		dlg := dialog.NewFolderOpen(func(lu fyne.ListableURI, err error) {
			if err == nil && lu != nil {
				log.Infof("open path %s", lu.Path())
				outputPath.Set(lu.Path())
			}
		}, a.window)
		dlg.Resize(
			fyne.NewSize(
				fyne.Min(a.window.Canvas().Size().Width, 740),
				fyne.Min(a.window.Canvas().Size().Height, 580),
			),
		)
		dlg.Show()
	})

	overwriteOption := binding.NewBool()
	ripTexOption := binding.NewBool()
	thumbnailsOption := binding.NewBool()

	overwriteCheck := widget.NewCheckWithData(lang.X("unpack.option.overwrite.text", "Overwrite existing files"), overwriteOption)
	ripTexCheck := widget.NewCheckWithData(lang.X("unpack.option.repTex.text", "Rip Textures to Png"), ripTexOption)
	thumbnailsCheck := widget.NewCheckWithData(lang.X("unpack.option.thumbnails.text", "Extract thumbnails"), thumbnailsOption)

	extrctBtn := widget.NewButtonWithIcon(lang.X("unpack.extractBtn.text", "Extract"), theme.UploadIcon(), func() {
		path, err := outputPath.Get()
		if err != nil {
			log.WithError(err).Error("error collecting bound output path value")
			return
		}
		overwrite, err := overwriteOption.Get()
		if err != nil {
			log.WithError(err).Error("error collecting bound overwrite value")
			return
		}
		ripTex, err := ripTexOption.Get()
		if err != nil {
			log.WithError(err).Error("error collecting bound ripTex value")
			return
		}
		thumbnails, err := thumbnailsOption.Get()
		if err != nil {
			log.WithError(err).Error("error collecting bound thumbnails value")
			return
		}
		a.extractPackage(path, ddpackage.UnpackOptions{
			Overwrite:   overwrite,
			RipTextures: ripTex,
			Thumbnails:  thumbnails,
		})
	})

	extractForm := container.NewVBox(
		container.New(
			NewLeftExpandHBoxLayout(),
			outEntry,
			outBrowseBtn,
		),
		container.NewHBox(
			overwriteCheck,
			ripTexCheck,
			thumbnailsCheck,
		),
		extrctBtn,
	)

	log.Info("form built")

	disableButtonsListener := binding.NewDataListener(func() {
		disable, _ := a.disableButtons.Get()
		log.Info("unpack buttons disable: ", disable)
		if disable {
			outBrowseBtn.Disable()
			extrctBtn.Disable()
		} else {
			outBrowseBtn.Enable()
			extrctBtn.Enable()
		}
	})

	log.Info("listen built")

	a.setMainContent(
		container.New(
			NewBottomExpandVBoxLayout(),
			extractForm,
			split,
		),
		disableButtonsListener,
	)

	log.Info("main content set")

	a.disableButtons.Set(false)
}

func (a *App) buildPackageTree() (*widget.Tree, binding.String, map[string]*structures.FileInfo) {

	nodeTree, nodeMap := buildInfoMaps(&a.pkg.FileList)

	log.Debugf("nodeTree: %s", spew.Sdump(nodeTree))

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
			_, ok := nodeMap[tni]
			return !ok
		},
		func(b bool) fyne.CanvasObject {
			return widget.NewLabel("label template")
		},
		func(tni widget.TreeNodeID, b bool, co fyne.CanvasObject) {
			_, file := filepath.Split(tni)
			if b {
				co.(*widget.Label).SetText(file + "/")
			} else {
				co.(*widget.Label).SetText(file)
			}
		},
	)

	tree.OnSelected = func(uid widget.TreeNodeID) {
		selected.Set(uid)
	}

	return tree, selected, nodeMap
}

func buildInfoMaps(infoList *[]structures.FileInfo) (map[string][]string, map[string]*structures.FileInfo) {
	nodeMap := make(map[string]*structures.FileInfo, len(*infoList))
	nodeTree := make(map[string][]string)
	for i := 0; i < len(*infoList); i++ {
		info := &(*infoList)[i]
		if strings.HasPrefix(info.RelPath, "thumbnails/") {
			continue
		}
		if strings.HasSuffix(info.RelPath, ".json") {
			continue
		}
		nodeMap[info.RelPath] = info

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

	return nodeTree, nodeMap
}

func (a *App) extractPackage(path string, options ddpackage.UnpackOptions) {
	a.disableButtons.Set(true)
	progressVal := binding.NewFloat()
	progressBar := widget.NewProgressBarWithData(progressVal)

	targetPath := filepath.Join(path, a.pkg.Name())
	progressDlg := dialog.NewCustomWithoutButtons(
		lang.X("unpack.extractProgressDlg.title", "Extracting to {{.Path}}", map[string]any{"Path": targetPath}),
		progressBar,
		a.window,
	)
	progressDlg.Show()
	go func() {
		err := a.pkg.ExtractPackage(a.pkgFile, path, options, func(p, t int) {
			progressVal.Set(float64(p) / float64(t))
		})
		progressDlg.Hide()
		packPath, _ := a.operatingPath.Get()
		if err != nil {
			errDlg := dialog.NewError(
				errors.Join(err, errors.New(lang.X(
					"unpack.ununpack.error.text",
					"Error unpcking {{.Pack}} to {{.Path}}",
					map[string]any{
						"Pack": packPath,
						"Path": targetPath,
					},
				))),
				a.window,
			)
			errDlg.Show()
			return
		}
		infoDlg := dialog.NewInformation(
			lang.X("unpack.success.dlg.title", "Extraction successful"),
			lang.X(
				"unpack.success.dlg.text",
				"{{.Pack}} extracted to {{.Path}} successfully",
				map[string]any{
					"Pack": packPath,
					"Path": targetPath,
				}),
			a.window,
		)
		infoDlg.Show()
		a.disableButtons.Set(false)
	}()
}
