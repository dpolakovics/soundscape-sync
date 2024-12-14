package ui

import (
    _ "embed"
    "net/url"
    "os/exec"
    "path/filepath"
    "runtime"
    "strings"

    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/dialog"
    "fyne.io/fyne/v2/widget"

    "github.com/dpolakovics/soundscape-sync/internal/logic"
)

//go:embed bmc.png
var bmcPng []byte

// tryLinuxNativeFolderDialog attempts to open a native OS folder selection dialog on Linux.
// It first tries zenity (common on GNOME), then kdialog (common on KDE).
// If neither works, it returns an empty string.
func tryLinuxNativeFolderDialog() string {
    // Try zenity first
    cmd := exec.Command("zenity", "--file-selection", "--directory")
    out, err := cmd.Output()
    if err == nil {
        folderPath := strings.TrimSpace(string(out))
        if folderPath != "" {
            return folderPath
        }
    }

    // If zenity fails, try kdialog
    cmd = exec.Command("kdialog", "--getexistingdirectory", "$HOME")
    out, err = cmd.Output()
    if err == nil {
        folderPath := strings.TrimSpace(string(out))
        if folderPath != "" {
            return folderPath
        }
    }

    // If both fail, return empty string
    return ""
}

// tryNativeFolderDialog attempts to open a native OS folder selection dialog.
// If successful, it returns the selected path. If not available or fails, it returns an empty string.
// On Windows, use an OpenFileDialog trick to simulate a folder selection with no visible cmd window.
func tryNativeFolderDialog() string {
    switch runtime.GOOS {
    case "windows":
        cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
            "[System.Reflection.Assembly]::LoadWithPartialName('System.Windows.Forms') | Out-Null; "+
                "$ofd = New-Object System.Windows.Forms.OpenFileDialog; "+
                "$ofd.InitialDirectory = [Environment]::GetFolderPath('MyDocuments'); "+
                "$ofd.ValidateNames = $true; $ofd.CheckFileExists = $false; $ofd.CheckPathExists = $true; "+
                "$ofd.FileName = 'Folder Selection.'; "+
                "if ($ofd.ShowDialog() -eq 'OK') { Split-Path $ofd.FileName }")

        if runtime.GOOS == "windows" {
            cmd.SysProcAttr = getSysProcAttr()
        }

        out, err := cmd.Output()
        if err == nil {
            folderPath := strings.TrimSpace(string(out))
            if folderPath != "" {
                return folderPath
            }
        }
        return ""

    case "darwin":
        // On macOS, use AppleScript to choose a folder
        cmd := exec.Command("osascript", "-e", `tell application "System Events" to activate`, "-e", `POSIX path of (choose folder)`)
        out, err := cmd.Output()
        if err == nil {
            folderPath := strings.TrimSpace(string(out))
            if folderPath != "" {
                return folderPath
            }
        }
        return ""

    case "linux":
        return tryLinuxNativeFolderDialog()

    default:
        return ""
    }
}

// showFolderSelection attempts to show a native file dialog first. If it fails, fallback to Fyne's dialog.
func showFolderSelection(win fyne.Window, callback func(string)) {
    // Try native first
    nativePath := tryNativeFolderDialog()
    if nativePath != "" {
        callback(nativePath)
        return
    }

    // Fallback to Fyne dialog if native is not available
    dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
        if err != nil {
            dialog.ShowError(err, win)
            return
        }
        if uri == nil {
            return
        }
        callback(uri.Path())
    }, win)
}

func CreateMainContent(app fyne.App, window fyne.Window) fyne.CanvasObject {
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

    // Set up folder selection actions with fallback logic
    folder1Button.OnTapped = func() {
        showFolderSelection(window, func(path string) {
            folder1 = path
            folder1Label.SetText(filepath.Base(path))
            updateStartButton(folder1Label, folder2Label, folderOutputLabel, startButton)
        })
    }

    folder2Button.OnTapped = func() {
        showFolderSelection(window, func(path string) {
            folder2 = path
            folder2Label.SetText(filepath.Base(path))
            updateStartButton(folder1Label, folder2Label, folderOutputLabel, startButton)
        })
    }

    folderOutputButton.OnTapped = func() {
        showFolderSelection(window, func(path string) {
            folderOutput = path
            folderOutputLabel.SetText(filepath.Base(path))
            updateStartButton(folder1Label, folder2Label, folderOutputLabel, startButton)
        })
    }

    // Volume slider
    volumeSliderValueLabel := widget.NewLabel("Adjust Soundscape volume")
    // Create the slider
    volumeSlider := widget.NewSlider(0, 100)
    volumeSlider.Step = 1
    volumeSlider.Value = 100

    // Set up start button action
    startButton.OnTapped = func() {
        startButton.Disable()
        progressBar.Show()
        go func() {
            err := logic.CombineFiles(folder1, folder2, folderOutput, progressBar, volumeSlider.Value)
            if err != nil {
                dialog.ShowError(err, window)
            } else {
                dialog.ShowInformation("Success", "Audio files combined successfully", window)
            }
            progressBar.Hide()
            startButton.Enable()
        }()
    }

    // Create an image for the Buy Me a Coffee button
    bmcResource := fyne.NewStaticResource("bmc.png", bmcPng)
    // Create the Buy Me a Coffee button with an image
    bmcButton := widget.NewButtonWithIcon("Buy me a coffee", bmcResource, func() {
        u, _ := url.Parse("https://www.buymeacoffee.com/razormind")
        _ = app.OpenURL(u)
    })

    // Append new text
    newText := "I am an individual developer who has created an app for Soundscape synchronization."
    newText = newText + "\nI hope this app helps you as much as it has helped me."
    newText = newText + "\nIf you find it useful, please consider buying me a coffee. Thank you!"
    multiLineEntry := widget.NewMultiLineEntry()
    multiLineEntry.SetText(newText)

    // Create and return the main content
    return container.NewVBox(
        container.NewHBox(folder1Button, folder1Label),
        container.NewHBox(folder2Button, folder2Label),
        container.NewHBox(folderOutputButton, folderOutputLabel),
        volumeSliderValueLabel,
        volumeSlider,
        startButton,
        multiLineEntry,
        bmcButton,
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