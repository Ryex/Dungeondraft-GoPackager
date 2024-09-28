package gui

import (
	"path/filepath"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/bindings"
	"github.com/ryex/dungeondraft-gopackager/internal/gui/widgets"
	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/ddpackage"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
)

type PackJSONDialog struct {
	Path string

	Name     string
	Author   string
	Version  string
	Keywords []string

	ID string

	Allow3rdParty     bool
	sourceHas3rdParty bool

	ColorOverrides *structures.CustomColorOverrides

	content *fyne.Container
	parent  fyne.Window
	dialog  *dialog.CustomDialog

	callback func(options ddpackage.SavePackageJSONOptions)

	editable bool
}

func NewPackJSONDialogPkg(
	pkg *ddpackage.Package,
	editable bool,
	callback func(options ddpackage.SavePackageJSONOptions),
	parent fyne.Window,
) *PackJSONDialog {
	dlg := &PackJSONDialog{
		parent:   parent,
		editable: editable,
		callback: callback,
	}
	if pkg != nil {
		dlg.ID = pkg.ID()
		dlg.Path = pkg.UnpackedPath()
		dlg.Name = pkg.Name()
		dlg.Author = pkg.Info().Author
		dlg.Version = pkg.Info().Version
		keywords := make([]string, len(pkg.Info().Keywords))
		copy(keywords, pkg.Info().Keywords)
		dlg.Keywords = keywords
		if pkg.Info().Allow3rdParty != nil {
			dlg.sourceHas3rdParty = true
			dlg.Allow3rdParty = *pkg.Info().Allow3rdParty
		}
		overrides := pkg.Info().ColorOverrides

		dlg.ColorOverrides = &overrides
	}
	if dlg.ColorOverrides == nil {
		dlg.ColorOverrides = structures.DefaultCustomColorOverrides()
	}
	if dlg.Version == "" {
		dlg.Version = "1"
	}
	dlg.buildUi()
	return dlg
}

func NewPackJSONDialog(
	path string,
	callback func(options ddpackage.SavePackageJSONOptions),
	parent fyne.Window,
) *PackJSONDialog {
	dlg := &PackJSONDialog{
		Path:           path,
		Name:           filepath.Base(filepath.Dir(path)),
		ID:             ddpackage.GenPackID(),
		parent:         parent,
		callback:       callback,
		Version:        "1",
		ColorOverrides: structures.DefaultCustomColorOverrides(),
		editable:       true,
	}
	dlg.buildUi()
	return dlg
}

func (dlg *PackJSONDialog) buildUi() {
	IDLbl := widget.NewLabel(lang.X("packJson.id.label", "ID"))
	IDEntry := widget.NewEntryWithData(binding.BindString(&dlg.ID))

	nameLbl := widget.NewLabel(lang.X("packJson.name.label", "Name"))
	nameEntry := widget.NewEntryWithData(binding.BindString(&dlg.Name))
	nameEntry.SetPlaceHolder(lang.X("packJson.name.placeholder", "Package name"))

	authorLbl := widget.NewLabel(lang.X("packJson.author.label", "Author"))
	authorEntry := widget.NewEntryWithData(binding.BindString(&dlg.Author))
	authorEntry.SetPlaceHolder(lang.X("packJson.author.placeholder", "Package author"))

	versionLbl := widget.NewLabel(lang.X("packJson.version.label", "Version"))
	versionEntry := widgets.NewSpinner(1, 10000, 0.1)
	versionEntry.Bind(bindings.NewReversableMapping(
		binding.BindString(&dlg.Version),
		func(ver string) (float64, error) {
			return strconv.ParseFloat(ver, 64)
		}, func(val float64) (string, error) {
			return strconv.FormatFloat(val, 'f', -1, 64), nil
		},
	))

	keywordsLbl := widget.NewLabel(lang.X("packJson.keywords.label", "Keywords"))
	keywordsEntry := widget.NewEntryWithData(bindings.NewReversableMapping(
		binding.BindStringList(&dlg.Keywords),
		func(list []string) (string, error) {
			return strings.Join(list, ","), nil
		},
		func(csv string) ([]string, error) {
			words := utils.Map(strings.Split(csv, ","), func(s string) string {
				return strings.TrimSpace(s)
			})
			words = utils.Filter(words, func(s string) bool {
				return s != ""
			})
			return words, nil
		},
	))
	keywordsEntry.SetPlaceHolder(lang.X("packJson.keywords.placeholder", "split keywords with commas"))

	thirdPartyCheck := widget.NewCheck(lang.X("packJson.thirdParty.label", "Allow 3rd party mapping software to use this pack"), func(checked bool) {
		dlg.Allow3rdParty = checked
	})
	thirdPartyCheck.SetChecked(dlg.Allow3rdParty)

	customColorsMsg := multilineCanvasText(
		lang.X(
			"packJson.customColors.msg",
			"These settings affect the detection of the red color for custom color objects.\n"+
				"Most users will want to leave these settings alone.",
		),
		12,
		fyne.TextStyle{},
		fyne.TextAlignLeading,
		theme.Color(theme.ColorNameForeground),
	)

	minRednessLbl := widget.NewLabel(lang.X("packJson.minRedness.label", "Minimum Redness"))
	minRednessEntry := widgets.NewSpinner(-1, 1, 0.001)
	minRednessEntry.Bind(binding.BindFloat(&dlg.ColorOverrides.MinRedness))

	minSaturationLbl := widget.NewLabel(lang.X("packJson.minSaturation.label", "Minimum Saturation"))
	minSaturationEntry := widgets.NewSpinner(-1, 1, 0.001)
	minSaturationEntry.Bind(binding.BindFloat(&dlg.ColorOverrides.MinSaturation))

	rednessToleranceLbl := widget.NewLabel(lang.X("packJson.redTolerance.label", "Redness Tolerance"))
	rednessToleranceEntry := widgets.NewSpinner(-1, 1, 0.001)
	rednessToleranceEntry.Bind(binding.BindFloat(&dlg.ColorOverrides.RedTolerance))

	customColorsContainer := container.New(
		layout.NewFormLayout(),
		minRednessLbl, minRednessEntry,
		minSaturationLbl, minSaturationEntry,
		rednessToleranceLbl, rednessToleranceEntry,
	)

	if !dlg.ColorOverrides.Enabled {
		customColorsContainer.Hide()
	}

	customColorsCheck := widget.NewCheck(lang.X("packJson.customColors.enable.label", "Override custom color settings"), func(checked bool) {
		dlg.ColorOverrides.Enabled = checked
		if checked {
			customColorsContainer.Show()
		} else {
			customColorsContainer.Hide()
		}
	})
	customColorsCheck.SetChecked(dlg.ColorOverrides.Enabled)

	if !dlg.editable {
		IDEntry.Disable()
		nameEntry.Disable()
		authorEntry.Disable()
		versionEntry.Disable()
		keywordsEntry.Disable()
		thirdPartyCheck.Disable()
		customColorsCheck.Disable()
		minRednessEntry.Disable()
		minSaturationEntry.Disable()
		rednessToleranceEntry.Disable()
	}

	dlg.content = container.NewVBox(
		container.New(
			layout.NewFormLayout(),
			IDLbl, IDEntry,

			nameLbl, nameEntry,
			authorLbl, authorEntry,
			versionLbl, versionEntry,
			keywordsLbl, keywordsEntry,
		),
		thirdPartyCheck,
		customColorsCheck,
		customColorsMsg,
		customColorsContainer,
	)

	dlg.dialog = dialog.NewCustom(
		lang.X("packJson.dialog.title", "Package settings"),
		lang.X("packJson.dismis", "Close"),
		dlg.content,
		dlg.parent,
	)

	defaultButtons := []fyne.CanvasObject{
		widget.NewButton(lang.X("packJson.dialog.dismis", "Cancel"), dlg.dialog.Hide),
		widget.NewButtonWithIcon(lang.X("packJson.dialog.save", "Save"), theme.DocumentSaveIcon(), dlg.onSave),
	}

	if dlg.editable {
		dlg.dialog.SetButtons(defaultButtons)
	}
}

func (dlg *PackJSONDialog) onSave() {
	dlg.dialog.Hide()
	options := ddpackage.SavePackageJSONOptions{
		Path:          dlg.Path,
		Name:          dlg.Name,
		ID:            dlg.ID,
		Author:        dlg.Author,
		Version:       dlg.Version,
		Keywords:      dlg.Keywords,
		ColorOverides: *dlg.ColorOverrides,
	}
	if dlg.Allow3rdParty || dlg.sourceHas3rdParty {
		allow := dlg.Allow3rdParty
		options.Allow3rdParty = &allow
	}
	if dlg.callback != nil {
		dlg.callback(options)
	}
}

func (dlg *PackJSONDialog) Show() {
	dlg.dialog.Show()
}

func (dlg *PackJSONDialog) Hide() {
	dlg.dialog.Hide()
}
