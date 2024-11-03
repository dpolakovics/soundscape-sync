package ui

import (
    "path/filepath"

    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/dialog"
    "fyne.io/fyne/v2/widget"

    "soundscape-sync/internal/logic"
)

func CreateMainContent(window fyne.Window) fyne.CanvasObject {
    var folder1, folder2, folderOutput string
    // Create folder selection buttons
    folder1Button := widget.NewButton("Select the Folder with the Soundscape", nil)
    folder2Button := widget.NewButton("Select the Folder with the Audiobook", nil)
    folderOutputButton := widget.NewButton("Select the output Folder", nil)

    // Create labels to display selected folders
    folder1Label := widget.NewLabel("No folder selected")
    folder2Label := widget.NewLabel("No folder selected")
    folderOutputLabel := widget.NewLabel("No folder selected")

    // Create start button
    startButton := widget.NewButton("Start Sync", nil)
    startButton.Disable() // Disable initially

    // Create progress bar
    progressBar := widget.NewProgressBar()
    progressBar.Hide() // Hide initially

    // Set up folder selection dialogs
    folder1Button.OnTapped = func() {
        dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
            if err != nil {
                dialog.ShowError(err, fyne.CurrentApp().Driver().AllWindows()[0])
                return
            }
            if uri == nil {
                return
            }
            folder1 = uri.Path()
            folder1Label.SetText(filepath.Base(uri.Path()))
            updateStartButton(folder1Label, folder2Label, folderOutputLabel, startButton)
        }, fyne.CurrentApp().Driver().AllWindows()[0])
    }

    folder2Button.OnTapped = func() {
        dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
            if err != nil {
                dialog.ShowError(err, fyne.CurrentApp().Driver().AllWindows()[0])
                return
            }
            if uri == nil {
                return
            }
            folder2 = uri.Path()
            folder2Label.SetText(filepath.Base(uri.Path()))
            updateStartButton(folder1Label, folder2Label, folderOutputLabel, startButton)
        }, fyne.CurrentApp().Driver().AllWindows()[0])
    }

    folderOutputButton.OnTapped = func() {
        dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
            if err != nil {
                dialog.ShowError(err, fyne.CurrentApp().Driver().AllWindows()[0])
                return
            }
            if uri == nil {
                return
            }
            folderOutput = uri.Path()
            folderOutputLabel.SetText(filepath.Base(uri.Path()))
            updateStartButton(folder1Label, folder2Label, folderOutputLabel, startButton)
        }, fyne.CurrentApp().Driver().AllWindows()[0])
    }

    // Set up start button action
    startButton.OnTapped = func() {
        startButton.Disable()
        progressBar.Show()
        go func() {
            err := logic.CombineFiles(folder1, folder2, folderOutput, progressBar)
            if err != nil {
                dialog.ShowError(err, window)
            } else {
                dialog.ShowInformation("Success", "Audio files combined successfully", window)
            }
            progressBar.Hide()
            startButton.Enable()
        }()
    }

    // Create and return the main content
    return container.NewVBox(
        container.NewHBox(folder1Button, folder1Label),
        container.NewHBox(folder2Button, folder2Label),
        container.NewHBox(folderOutputButton, folderOutputLabel),
        startButton,
        progressBar,
    )
}

func updateStartButton(label1, label2, folderOutputLabel *widget.Label, button *widget.Button) {
    if label1.Text != "No folder selected" && label2.Text != "No folder selected" && folderOutputLabel.Text != "No folder selected" {
        button.Enable()
    } else {
        button.Disable()
    }
}
