package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	cLayout "fyne.io/x/fyne/layout"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/bindings"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/layouts"
	log "github.com/sirupsen/logrus"
)

func (a *App) createTagSetsDialog(editable bool) dialog.Dialog {
	var (
		tags    []string
		tagSets []string
	)

	boundTags := binding.BindStringList(&tags)
	boundTagSets := binding.BindStringList(&tagSets)
	curSet := binding.NewString()

	updateTags := func() {
		set, err := curSet.Get()
		if err != nil {
			log.WithError(err).Error("error getting bound set value")
			return
		}
		ts := a.pkg.Tags().Set(set)
		var updated []string
		if ts != nil {
			updated = ts.AsSlice()
		}
		boundTags.Set(updated)
	}

	updateSets := func() {
		updated := a.pkg.Tags().AllSets()
		boundTagSets.Set(updated)
	}
	updateSets()

	curSet.AddListener(binding.NewDataListener(updateTags))

	tagSetList := widget.NewListWithData(
		boundTagSets,
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
					set, err := di.(binding.String).Get()
					if err != nil {
						log.WithError(err).Errorf("failed to get set in del set btn")
						return
					}
					a.pkg.Tags().DeleteSet(set)
					updateSets()
					a.saveUnpackedTags()
				}
			} else {
				btn.Disable()
				btn.Hide()
			}
		},
	)

	tagSetList.OnSelected = func(id widget.ListItemID) {
		curSet.Set(tagSets[id])
	}

	tagList := widget.NewListWithData(
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
						log.WithError(err).Error("failed to get tag in rm tag from set btn")
						return
					}
					set, err := curSet.Get()
					if err != nil {
						log.WithError(err).Errorf("failed to get set in rm tag from set btn for tag %s", tag)
						return
					}
					a.pkg.Tags().RemoveTagFromSet(set, tag)
					updateTags()
					a.saveUnpackedTags()
				}
			} else {
				btn.Disable()
				btn.Hide()
			}
		},
	)

	tagSetHeader := container.NewStack(
		&canvas.Rectangle{
			FillColor:    theme.Color(theme.ColorNameHeaderBackground),
			CornerRadius: 4,
		},
		container.NewPadded(
			widget.NewLabel(
				lang.X("tagSets.tagSet.label.text", "Tag Sets"),
			),
		),
	)

	tagsHeader := container.NewStack(
		&canvas.Rectangle{
			FillColor:    theme.Color(theme.ColorNameHeaderBackground),
			CornerRadius: 4,
		},
		container.NewPadded(
			widget.NewLabelWithData(
				bindings.NewMapping(
					curSet,
					func(set string) (string, error) {
						return lang.X(
							"tagSets.tagsFor.label.text",
							"Tags for Set: {{.Set}}",
							map[string]any{
								"Set": set,
							},
						), nil
					},
				),
			),
		),
	)

	setSide := layouts.NewTopExpandVBox(
		layouts.NewBottomExpandVBox(
			tagSetHeader,
			container.NewStack(
				&canvas.Rectangle{
					FillColor:    theme.Color(theme.ColorNameInputBackground),
					CornerRadius: 4,
				},
				tagSetList,
			),
		),
	)

	tagsSide := layouts.NewTopExpandVBox(
		layouts.NewBottomExpandVBox(
			tagsHeader,
			container.NewStack(
				&canvas.Rectangle{
					FillColor:    theme.Color(theme.ColorNameInputBackground),
					CornerRadius: 4,
				},
				tagList,
			),
		),
	)

	if editable {

		setNameEntry := widget.NewEntry()
		setAddBtn := widget.NewButtonWithIcon(
			lang.X("tagSets.setAddBtn.text", "Add"),
			theme.ContentAddIcon(),
			func() {
				if setNameEntry.Text == "" {
					return
				}
				a.pkg.Tags().AddSet(setNameEntry.Text)
				updateSets()
				a.saveUnpackedTags()
				setNameEntry.SetText("")
			},
		)
		setSide.Add(layouts.NewLeftExpandHBox(setNameEntry, setAddBtn))

		tagSelector := widget.NewSelectEntry(a.pkg.Tags().AllTags())
		tagAddBtn := widget.NewButtonWithIcon(
			lang.X("tagSets.tagAddBtn.text", "Add"),
			theme.ContentAddIcon(),
			func() {
				if tagSelector.Text == "" {
					return
				}
				set, err := curSet.Get()
				if err != nil {
					log.WithError(err).Error("failed to get set in add tag to set btn")
					return
				}
				a.pkg.Tags().AddTag(tagSelector.Text)
				a.pkg.Tags().AddTagToSet(set, tagSelector.Text)
				updateTags()
				a.saveUnpackedTags()
				tagSelector.SetText("")
				tagSelector.SetOptions(a.pkg.Tags().AllTags())
			},
		)
		tagsSide.Add(layouts.NewLeftExpandHBox(tagSelector, tagAddBtn))

	}

	content := container.NewPadded(
		container.New(
			cLayout.NewHPortion([]float64{50, 0.1, 50}),
			setSide,
			widget.NewSeparator(),
			tagsSide,
		),
	)

	dlg := dialog.NewCustom(
		lang.X("tagSets.dialog.title", "Tag Sets"),
		lang.X("tagSets.dialog.dismiss", "Close"),
		content,
		a.window,
	)
	dlg.Resize(
		fyne.NewSize(
			fyne.Min(a.window.Canvas().Size().Width, 740),
			fyne.Min(a.window.Canvas().Size().Height, 580),
		),
	)
	content.Refresh()
	return dlg
}
