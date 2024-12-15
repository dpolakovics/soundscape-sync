package main

import (
    "context"
    "net/url"
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
	currentTag = "v0.10"
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
		dialog.ShowError(err, w)
		return
	}

	latestTag := release.GetTagName()
	if latestTag > currentTag {
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
