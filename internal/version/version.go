// Package version reports the version and the revision of the application
// build.
//
// The revision is read from the build information that the go tool embeds into
// the binary. Note that `go run` does not stamp it by default, because it loads
// packages without the automatic version control lookup; use
// `go run -buildvcs=true` to see the revision in a development build.
package version

import (
	"runtime/debug"
)

// shortCommitLength is the number of characters of the commit hash that is
// shown to the user, the length `git rev-parse --short` uses.
const shortCommitLength = 7

// Version overrides the version the application reports itself when it is not
// set from the application metadata. A build can set it, e.g. for the CLI:
//
//	go build -ldflags "-X github.com/ryex/dungeondraft-gopackager/internal/version.Version=2.1.0"
var Version string

// Commit is the git revision the application was built from.
//
// The go tool embeds the revision of the repository the binary is built from
// (since go 1.18), so it is read from the build information and no git command
// has to run during the build. A build can override it, e.g. when it is not
// built from a checkout:
//
//	go build -ldflags "-X github.com/ryex/dungeondraft-gopackager/internal/version.Commit=$(git rev-parse --short HEAD)"
var Commit string

// Dirty reports whether the working tree had uncommitted changes when the
// application was built.
var Dirty bool

func init() {
	commit, dirty := vcs()
	if Commit == "" {
		Commit = commit
	}
	Dirty = dirty
}

// Display returns the version to show to the user: the version of the build,
// amended with the commit it was built from, e.g. "2.1.0 (1a2b3c4)".
//
// appVersion is the version the application reports itself, e.g. the one of
// the Fyne app metadata. It is used unless the version was set by the build.
func Display(appVersion string) string {
	if Version != "" {
		appVersion = Version
	}

	commit := shortCommit(Commit)
	if commit == "" {
		return appVersion
	}
	if Dirty {
		commit += "-dirty"
	}
	if appVersion == "" {
		return commit
	}

	return appVersion + " (" + commit + ")"
}

// vcs returns the revision and the modified state of the working tree that the
// go tool embedded into the binary. Both are empty when the binary carries no
// version control information, e.g. when it was built from a source archive.
func vcs() (commit string, dirty bool) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", false
	}

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			commit = setting.Value
		case "vcs.modified":
			dirty = setting.Value == "true"
		}
	}

	return commit, dirty
}

// shortCommit shortens the commit hash to the length git shows it in.
func shortCommit(commit string) string {
	if len(commit) > shortCommitLength {
		return commit[:shortCommitLength]
	}
	return commit
}
