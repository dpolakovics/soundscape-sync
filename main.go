package main

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/dpolakovics/soundscape-sync/internal/ui"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/google/go-github/v39/github"
)

const (
	owner      = "dpolakovics"
	repo       = "soundscape-sync"
	currentTag = "v1.1.1"
)

func main() {
	myApp := app.New()
	mainWindow := myApp.NewWindow("Soundscape Sync")

	content := ui.CreateMainContent(myApp, mainWindow)
	mainWindow.SetContent(content)

	checkForUpdates(myApp, mainWindow)

	mainWindow.Resize(fyne.NewSize(800, 600))
	mainWindow.ShowAndRun()
}

func checkForUpdates(a fyne.App, w fyne.Window) {
	client := github.NewClient(nil)
	release, _, err := client.Repositories.GetLatestRelease(context.Background(), owner, repo)
	if err != nil {
		showErrorDialog(w, fmt.Errorf("Failed to check for the latest release: %w. Please ensure you have an internet connection, and if the issue persists, reach out to the developer.", err))
		return
	}

	latestTag := release.GetTagName()
	fmt.Println("Latest tag:", latestTag)
	if isNewerVersion(currentTag, latestTag) {
		updateDialog := dialog.NewCustom("Update Available", "Close",
			container.NewVBox(
				widget.NewLabel("A new version is available!"),
				widget.NewButton("Open Release Page", func() {
					u, _ := url.Parse(release.GetHTMLURL())
					_ = a.OpenURL(u)
				}),
			), w)
		updateDialog.Resize(fyne.NewSize(300, 150))
		updateDialog.Show()
	}
}

func isNewerVersion(oldVer, newVer string) bool {
	oldVer = strings.TrimPrefix(oldVer, "v")
	newVer = strings.TrimPrefix(newVer, "v")

	oldParts := strings.Split(oldVer, ".")
	newParts := strings.Split(newVer, ".")

	for i := 0; i < len(oldParts) && i < len(newParts); i++ {
		oldNum, _ := strconv.Atoi(oldParts[i])
		newNum, _ := strconv.Atoi(newParts[i])
		if newNum > oldNum {
			return true
		} else if newNum < oldNum {
			return false
		}
	}

	return len(newParts) > len(oldParts)
}

func showErrorDialog(win fyne.Window, err error) {
	if err == nil {
		return
	}

	errorStr := err.Error()
	copyButton := widget.NewButton("Copy Error", func() {
		win.Clipboard().SetContent(errorStr)
	})

	errorLabel := widget.NewLabel(errorStr)
	errorLabel.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(errorLabel, copyButton)
	d := dialog.NewCustom("Error", "Close", content, win)
	d.Resize(fyne.NewSize(700, 500))
	d.Show()
}
