package main

import (
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/app"
    "soundscape-sync/internal/ui"
)

func main() {
    myApp := app.New()
    mainWindow := myApp.NewWindow("Soundscape Sync")

    content := ui.CreateMainContent(mainWindow)
    mainWindow.SetContent(content)

    mainWindow.Resize(fyne.NewSize(800, 600))
    mainWindow.ShowAndRun()
}
