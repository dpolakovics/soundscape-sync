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
    "fmt"
    "strings"
    "strconv"
)

const (
    owner      = "dpolakovics"
    repo       = "soundscape-sync"
    currentTag = "v0.9"
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
    // Strip leading 'v'
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

    // If all matched so far, then a longer new version means it's newer
    return len(newParts) > len(oldParts)
}