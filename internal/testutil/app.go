package testutil

import (
	"fyne.io/fyne/v2/test"
)

// SetupFyneApp creates a software rendered Fyne application for tests and
// installs it as the current application.
//
// Fyne's data binding package dispatches every notification through fyne.Do(),
// which panics when no application is running. Call this before creating or
// modifying any binding.
func SetupFyneApp() {
	test.NewApp()
}
