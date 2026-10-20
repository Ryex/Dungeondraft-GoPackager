// Package testutil provides helpers to create Dungeondraft resource pack
// fixtures on disk so that tests can exercise the packager and the GUI
// without needing binary sample packs in the repository.
package testutil

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryex/dungeondraft-gopackager/pkg/structures"
	ddcolor "github.com/ryex/dungeondraft-gopackager/pkg/structures/color"
)

// Fixture describes an unpacked Dungeondraft resource pack.
type Fixture struct {
	// ID is the unique pack id. It is used to build the res:// paths.
	ID string
	// Name is the display name of the pack.
	Name string
	// Author is the pack author.
	Author string
	// Version is the pack version.
	Version string
	// Keywords are the pack keywords.
	Keywords []string

	// Objects are the file names of the object textures created under
	// textures/objects. Objects are the only taggable resources.
	Objects []string
	// Walls are the file names of the wall textures created under
	// textures/walls. Each one gets a matching metadata sidecar.
	Walls []string
	// Tilesets are the file names of the tileset textures created under
	// textures/tilesets. Each one gets a matching metadata sidecar.
	Tilesets []string

	// Tags maps a tag name to the pack relative resource paths it applies to.
	Tags map[string][]string
	// Sets maps a set name to the tag names it contains.
	Sets map[string][]string
}

// DefaultFixture returns a small fixture with two objects, a wall and a
// tileset, tagged with two tags collected in one set.
func DefaultFixture() *Fixture {
	return &Fixture{
		ID:       "testpack",
		Name:     "Test Pack",
		Author:   "Test Author",
		Version:  "1.0.0",
		Keywords: []string{"test"},
		Objects:  []string{"Rock", "Tree"},
		Walls:    []string{"Stone"},
		Tilesets: []string{"Grass"},
		Tags: map[string][]string{
			"nature": {"textures/objects/Tree.png"},
			"rock":   {"textures/objects/Rock.png"},
		},
		Sets: map[string][]string{
			"terrain": {"nature", "rock"},
		},
	}
}

// Write creates the fixture in a temporary directory and returns the path of
// the unpacked package.
func (f *Fixture) Write(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	f.WriteTo(t, dir)
	return dir
}

// WriteTo creates the fixture in the given directory.
func (f *Fixture) WriteTo(t *testing.T, dir string) {
	t.Helper()

	if f.ID == "" {
		f.ID = "testpack"
	}
	if f.Name == "" {
		f.Name = "Test Pack"
	}
	if f.Version == "" {
		f.Version = "1.0.0"
	}

	f.writePackJSON(t, dir)

	for _, name := range f.Objects {
		f.writeTexture(t, dir, "objects", name)
	}
	for _, name := range f.Walls {
		f.writeTexture(t, dir, "walls", name)
		f.writeWallMetadata(t, dir, name)
	}
	for _, name := range f.Tilesets {
		f.writeTexture(t, dir, "tilesets", name)
		f.writeTilesetMetadata(t, dir, name)
	}

	f.writeTags(t, dir)
}

func (f *Fixture) writePackJSON(t *testing.T, dir string) {
	t.Helper()

	allowThirdParty := true
	info := structures.PackageInfo{
		Name:          f.Name,
		ID:            f.ID,
		Version:       f.Version,
		Author:        f.Author,
		KeywordsRaw:   strings.Join(f.Keywords, ","),
		Allow3rdParty: &allowThirdParty,
	}

	data, err := json.MarshalIndent(&info, "", "  ")
	if err != nil {
		t.Fatalf("could not marshal pack.json: %s", err)
	}

	writeFile(t, filepath.Join(dir, "pack.json"), data)
}

func (f *Fixture) writeTags(t *testing.T, dir string) {
	t.Helper()

	tags := structures.NewPackageTags()
	for tag, resources := range f.Tags {
		tags.Tag(tag, resources...)
	}
	for set, tagNames := range f.Sets {
		tags.AddSet(set)
		tags.AddTagToSet(set, tagNames...)
	}

	data, err := json.MarshalIndent(tags, "", "  ")
	if err != nil {
		t.Fatalf("could not marshal tags: %s", err)
	}

	writeFile(t, filepath.Join(dir, "data", "default.dungeondraft_tags"), data)
}

func (f *Fixture) writeWallMetadata(t *testing.T, dir string, name string) {
	t.Helper()

	wall := structures.PackageWall{
		Path:  "textures/walls/" + name + ".png",
		Color: ddcolor.Color{R: 128, G: 128, B: 128, A: 255},
	}

	data, err := json.MarshalIndent(&wall, "", "  ")
	if err != nil {
		t.Fatalf("could not marshal wall metadata: %s", err)
	}

	writeFile(t, filepath.Join(dir, "data", "walls", name+".dungeondraft_wall"), data)
}

func (f *Fixture) writeTilesetMetadata(t *testing.T, dir string, name string) {
	t.Helper()

	tileset := structures.PackageTileset{
		Path:  "textures/tilesets/" + name + ".png",
		Name:  name,
		Type:  structures.TilesetNormal,
		Color: ddcolor.Color{R: 255, G: 255, B: 255, A: 255},
	}

	data, err := json.MarshalIndent(&tileset, "", "  ")
	if err != nil {
		t.Fatalf("could not marshal tileset metadata: %s", err)
	}

	writeFile(t, filepath.Join(dir, "data", "tilesets", name+".dungeondraft_tileset"), data)
}

// writeTexture creates a tiny opaque PNG so that the packer sees a real
// image of the right format.
func (f *Fixture) writeTexture(t *testing.T, dir string, kind string, name string) {
	t.Helper()

	path := filepath.Join(dir, "textures", kind, name+".png")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("could not create texture directory: %s", err)
	}

	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for x := range 4 {
		for y := range 4 {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 60), G: uint8(y * 60), B: 200, A: 255})
		}
	}

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("could not create texture %s: %s", path, err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		t.Fatalf("could not encode texture %s: %s", path, err)
	}
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("could not create directory for %s: %s", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("could not write %s: %s", path, err)
	}
}
