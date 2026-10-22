package version

import "testing"

func TestDisplay(t *testing.T) {
	tests := []struct {
		name       string
		appVersion string
		version    string
		commit     string
		dirty      bool
		want       string
	}{
		{
			name:       "the app version alone",
			appVersion: "2.1.0",
			want:       "2.1.0",
		},
		{
			name:       "the commit is shortened",
			appVersion: "2.1.0",
			commit:     "1a2b3c4d5e6f7a8b9c0d",
			want:       "2.1.0 (1a2b3c4)",
		},
		{
			name:       "an already short commit is kept",
			appVersion: "2.1.0",
			commit:     "1a2b3c4",
			want:       "2.1.0 (1a2b3c4)",
		},
		{
			name:       "a modified working tree is marked",
			appVersion: "2.1.0",
			commit:     "1a2b3c4d5e6f7a8b9c0d",
			dirty:      true,
			want:       "2.1.0 (1a2b3c4-dirty)",
		},
		{
			name:   "the commit alone when there is no app version",
			commit: "1a2b3c4d5e6f7a8b9c0d",
			want:   "1a2b3c4",
		},
		{
			name:       "a version set by the build wins over the app version",
			appVersion: "0.0.1",
			version:    "2.1.0",
			commit:     "1a2b3c4d5e6f7a8b9c0d",
			want:       "2.1.0 (1a2b3c4)",
		},
		{
			name:       "nothing at all",
			appVersion: "",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldVersion, oldCommit, oldDirty := Version, Commit, Dirty
			t.Cleanup(func() { Version, Commit, Dirty = oldVersion, oldCommit, oldDirty })

			Version, Commit, Dirty = tt.version, tt.commit, tt.dirty

			if got := Display(tt.appVersion); got != tt.want {
				t.Errorf("Display(%q) = %q, want %q", tt.appVersion, got, tt.want)
			}
		})
	}
}

func TestShortCommit(t *testing.T) {
	tests := []struct {
		commit string
		want   string
	}{
		{"", ""},
		{"1a2b3c4", "1a2b3c4"},
		{"1a2b3c4d5e6f7a8b9c0d", "1a2b3c4"},
	}

	for _, tt := range tests {
		if got := shortCommit(tt.commit); got != tt.want {
			t.Errorf("shortCommit(%q) = %q, want %q", tt.commit, got, tt.want)
		}
	}
}
