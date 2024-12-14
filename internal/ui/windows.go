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
    cmd := exec.Command("zenity", "--file-selection", "--directory")
    out, err := cmd.Output()
    if err == nil {
        folderPath := strings.TrimSpace(string(out))
        if folderPath != "" {
            return folderPath
        }
    }

    cmd = exec.Command("kdialog", "--getexistingdirectory", "$HOME")
    out, err = cmd.Output()
    if err == nil {
        folderPath := strings.TrimSpace(string(out))
        if folderPath != "" {
            return folderPath
        }
    }

    return ""
}

// tryNativeFolderDialog attempts to open a native OS folder selection dialog.
func tryNativeFolderDialog() string {
    switch runtime.GOOS {
    case "windows":
        cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command",
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
    nativePath := tryNativeFolderDialog()
    if nativePath != "" {
        callback(nativePath)
        return
    }

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

    folder1Button := widget.NewButton("Select the Folder with the Soundscape", nil)
    folder2Button := widget.NewButton("Select the Folder with the Audiobook", nil)
    folderOutputButton := widget.NewButton("Select the output Folder", nil)

    folder1Label := widget.NewLabel("No folder selected")
    folder2Label := widget.NewLabel("No folder selected")
    folderOutputLabel := widget.NewLabel("No folder selected")

    startButton := widget.NewButton("Start Sync", nil)
    startButton.Disable()

    progressBar := widget.NewProgressBar()
    progressBar.Hide()

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

    volumeSliderValueLabel := widget.NewLabel("Adjust Soundscape volume")
    volumeSlider := widget.NewSlider(0, 100)
    volumeSlider.Step = 1
    volumeSlider.Value = 100

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

    bmcResource := fyne.NewStaticResource("bmc.png", bmcPng)
    bmcButton := widget.NewButtonWithIcon("Buy me a coffee", bmcResource, func() {
        u, _ := url.Parse("https://www.buymeacoffee.com/razormind")
        _ = app.OpenURL(u)
    })

    newText := "I am an individual developer who has created an app for Soundscape synchronization."
    newText += "\nI hope this app helps you as much as it has helped me."
    newText += "\nIf you find it useful, please consider buying me a coffee. Thank you!"
    multiLineEntry := widget.NewMultiLineEntry()
    multiLineEntry.SetText(newText)

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