package cmd

import (
	"fmt"
	"slices"

	"github.com/ryex/dungeondraft-gopackager/internal/utils"
	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
)

type EditCmd struct {
	Pack EditPackCmd `cmd:"" help:"Edit Pack metadata (ID, Author, Name etc.)"`
	Tags EditTagsCmd `cmd:"" help:"Edit Tags for objects"`
	Sets EditSetsCmd `cmd:"" help:"Edit Tag Sets"`
}

type EditPackCmd struct {
	InputPath string `arg:"" type:"path" help:"the .dungeondraft_pack file or resource directory to work with"`

	ID      string `short:"I" help:"Unique ID for the pack"`
	Name    string `short:"N" help:"name of the package"`
	Author  string `short:"A" help:"package author"`
	Version string `short:"V" help:"package version"`

	AllowThirdParty *bool `short:"M" help:" set the 'allow_3rd_party_mapping_software_to_read' key. package will be incompatible with Dungeondraft v1.0.3.2" default:"true"`

	AddKeywords    []string `short:"AK" help:"comma separated keywords to add"`
	RemoveKeywords []string `short:"RK" help:"comma separated keywords to remove"`

	MinRedness    *float64 `short:"R" help:"enable custom colors and set the minimum redness value" default:"0.1"`
	MinSaturation *float64 `short:"S" help:"enable custom colors and set the minimum saturation value" default:"0"`
	RedTolerance  *float64 `short:"T" help:"enable custom colors and set the red tolerance value" default:"0.04"`
}

func (epc *EditPackCmd) Run(ctx *Context) error {
	err := ctx.LoadPkg(epc.InputPath)
	if err != nil {
		return err
	}

	if epc.ID != "" {
		ctx.Pkg.SetID(epc.ID)
	}

	if epc.Name != "" {
		ctx.Pkg.SetName(epc.Name)
	}

	if epc.Author != "" {
		ctx.Pkg.SetAuthor(epc.Author)
	}

	if epc.Version != "" {
		ctx.Pkg.SetVersion(epc.Version)
	}

	if len(epc.AddKeywords) > 0 {
		newkws := structures.SetFrom(ctx.Pkg.Info().Keywords)
		newkws.AddM(epc.AddKeywords...)
		ctx.Pkg.SetKeywords(newkws.AsSlice())
	}
	if len(epc.RemoveKeywords) > 0 {
		newkws := structures.SetFrom(ctx.Pkg.Info().Keywords)
		newkws.RemoveM(epc.RemoveKeywords...)
		ctx.Pkg.SetKeywords(newkws.AsSlice())
	}

	if epc.AllowThirdParty != nil {
		ctx.Pkg.SetAllow3rdParty(epc.AllowThirdParty)
	}

	overrides := ctx.Pkg.Info().ColorOverrides

	if epc.MinRedness != nil {
		overrides.MinRedness = *epc.MinRedness
	}

	if epc.MinSaturation != nil {
		overrides.MinSaturation = *epc.MinSaturation
	}

	if epc.RedTolerance != nil {
		overrides.RedTolerance = *epc.RedTolerance
	}

	ctx.Pkg.SetColorOverrides(overrides)

	err = ctx.Pkg.SaveUnpackedInfo()
	if err != nil {
		return err
	}

	return nil
}

type EditTagsCmd struct {
	InputPath string   `arg:"" type:"path" help:"the .dungeondraft_pack file or resource directory to work with"`
	Command   string   `arg:"" enum:"add,remove"`
	Tags      []string `short:"t" required:""`
	Globs     []string `arg:"" help:"patterns or paths to work with"`
}

func (etc *EditTagsCmd) Run(ctx *Context) error {
	err := ctx.LoadPkg(etc.InputPath)
	if err != nil {
		return err
	}
	err = ctx.LoadTags()
	if err != nil {
		return err
	}

	fileList, err := ctx.Pkg.FileList().Glob(func(fi *structures.FileInfo) bool {
		return fi.IsObject()
	}, etc.Globs...)
	if err != nil {
		return err
	}

	resPaths := slices.Collect(utils.Map(slices.Values(fileList), func(fi *structures.FileInfo) string {
		return fi.ResPath
	}))

	switch etc.Command {
	case "add":
		for _, tag := range etc.Tags {
			ctx.Pkg.Tags().Tag(tag, resPaths...)
		}
	case "remove":
		for _, tag := range etc.Tags {
			ctx.Pkg.Tags().Untag(tag, resPaths...)
		}
	default:
		return fmt.Errorf("unknown edit tags command '%s'", etc.Command)
	}

	err = ctx.Pkg.SaveUnpackedTags()
	if err != nil {
		return err
	}

	return nil
}

type EditSetsCmd struct {
	InputPath string   `arg:"" type:"path" help:"the .dungeondraft_pack file or resource directory to work with"`
	TagSet    string   `arg:"" help:"the tag Set to work with"`
	Command   string   `arg:"" enum:"add,remove"`
	Tags      []string `arg:"" help:"The tags to add or remove from the set"`
}

func (esc *EditSetsCmd) Run(ctx *Context) error {
	err := ctx.LoadPkg(esc.InputPath)
	if err != nil {
		return err
	}
	err = ctx.LoadTags()
	if err != nil {
		return err
	}

	switch esc.Command {
	case "add":
		ctx.Pkg.Tags().AddTagToSet(esc.TagSet, esc.Tags...)
	case "remove":
		ctx.Pkg.Tags().RemoveTagFromSet(esc.TagSet, esc.Tags...)
	default:
		return fmt.Errorf("unknown edit sets command '%s'", esc.Command)
	}

	err = ctx.Pkg.SaveUnpackedTags()
	if err != nil {
		return err
	}
	return nil
}
