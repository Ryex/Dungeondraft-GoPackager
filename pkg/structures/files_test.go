package structures

import "testing"

func TestIsWallEnd(t *testing.T) {
	cases := []struct {
		name    string
		resPath string
		want    bool
	}{
		{
			name:    "wall end piece",
			resPath: "res://packs/abc123/textures/walls/stone_end.png",
			want:    true,
		},
		{
			name:    "wall section",
			resPath: "res://packs/abc123/textures/walls/stone.png",
			want:    false,
		},
		{
			name:    "end in the middle of the stem",
			resPath: "res://packs/abc123/textures/walls/stone_end_long.png",
			want:    false,
		},
		{
			name:    "non wall texture with end suffix",
			resPath: "res://packs/abc123/textures/paths/road_end.png",
			want:    false,
		},
		{
			name:    "wall data file",
			resPath: "res://packs/abc123/data/walls/stone.dungeondraft_wall",
			want:    false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fi := &FileInfo{ResPath: c.resPath}
			if got := fi.IsWallEnd(); got != c.want {
				t.Errorf("IsWallEnd() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestShouldHaveMetadata(t *testing.T) {
	cases := []struct {
		name    string
		resPath string
		want    bool
	}{
		{
			name:    "wall section",
			resPath: "res://packs/abc123/textures/walls/stone.png",
			want:    true,
		},
		{
			name:    "wall end piece",
			resPath: "res://packs/abc123/textures/walls/stone_end.png",
			want:    false,
		},
		{
			name:    "tileset",
			resPath: "res://packs/abc123/textures/tilesets/floor.png",
			want:    true,
		},
		{
			name:    "terrain",
			resPath: "res://packs/abc123/textures/terrain/grass.png",
			want:    false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fi := &FileInfo{ResPath: c.resPath}
			if got := fi.ShouldHaveMetadata(); got != c.want {
				t.Errorf("ShouldHaveMetadata() = %v, want %v", got, c.want)
			}
		})
	}
}
