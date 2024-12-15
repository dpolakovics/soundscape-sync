package ui

import (
    _ "embed"
    "fmt"
    "net/url"
    "os/exec"
    "path/filepath"
    "runtime"
    "strings"

    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/dialog"
    "fyne.io/fyne/v2/layout"
    "fyne.io/fyne/v2/widget"

    "github.com/dpolakovics/soundscape-sync/internal/logic"
)

//go:embed bmc.png
var bmcPng []byte

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

    heading := widget.NewLabelWithStyle("Soundscape Sync", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

    folder1Label := widget.NewLabel("No folder selected")
    folder2Label := widget.NewLabel("No folder selected")
    folderOutputLabel := widget.NewLabel("No folder selected")

    startButton := widget.NewButton("Start Sync", nil)
    startButton.Disable()

    folder1Button := widget.NewButton("Select the Folder with the Soundscape", func() {
        showFolderSelection(window, func(path string) {
            folder1 = path
            folder1Label.SetText(filepath.Base(path))
            updateStartButton(folder1Label, folder2Label, folderOutputLabel, startButton)
        })
    })

    folder2Button := widget.NewButton("Select the Folder with the Audiobook", func() {
        showFolderSelection(window, func(path string) {
            folder2 = path
            folder2Label.SetText(filepath.Base(path))
            updateStartButton(folder1Label, folder2Label, folderOutputLabel, startButton)
        })
    })

    folderOutputButton := widget.NewButton("Select the output Folder", func() {
        showFolderSelection(window, func(path string) {
            folderOutput = path
            folderOutputLabel.SetText(filepath.Base(path))
            updateStartButton(folder1Label, folder2Label, folderOutputLabel, startButton)
        })
    })

    foldersCard := widget.NewCard(
        "Folder Selection",
        "Select input and output folders",
        container.NewVBox(
            container.NewHBox(folder1Button, folder1Label),
            container.NewHBox(folder2Button, folder2Label),
            container.NewHBox(folderOutputButton, folderOutputLabel),
        ),
    )

    volumeSliderValueLabel := widget.NewLabel("Volume: 100%")
    volumeSlider := widget.NewSlider(50, 100)
    volumeSlider.Step = 1
    volumeSlider.Value = 100
    volumeSlider.OnChanged = func(v float64) {
        volumeSliderValueLabel.SetText(fmt.Sprintf("Volume: %.0f%%", v))
    }

    sliderContainer := container.NewGridWrap(fyne.NewSize(400, volumeSlider.MinSize().Height), volumeSlider)

    volumeControls := container.NewVBox(
        volumeSliderValueLabel,
        container.NewHBox(
            sliderContainer,
            widget.NewLabel("(Default: 100%)"),
        ),
    )

    volumeCard := widget.NewCard(
        "Volume Settings",
        "Adjust the soundscape volume",
        volumeControls,
    )

    progressBar := widget.NewProgressBar()
    progressBar.Hide()

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

    actionCard := widget.NewCard(
        "",
        "",
        container.NewVBox(
            startButton,
            progressBar,
        ),
    )

    bmcResource := fyne.NewStaticResource("bmc.png", bmcPng)
    bmcButton := widget.NewButtonWithIcon("Buy me a coffee", bmcResource, func() {
        u, _ := url.Parse("https://www.buymeacoffee.com/razormind")
        _ = app.OpenURL(u)
    })

    newText := "I am an individual developer who has created an app for Soundscape synchronization.\n" +
        "I hope this app helps you as much as it has helped me.\n" +
        "If you find it useful, please consider buying me a coffee. Thank you!"
    
    // Use a label for static text to ensure consistent, visible color
    aboutLabel := widget.NewLabel(newText)
    aboutLabel.Wrapping = fyne.TextWrapWord

    supportCard := widget.NewCard(
        "About & Support",
        "",
        container.NewVBox(
            aboutLabel,
            bmcButton,
        ),
    )

    content := container.NewVBox(
        heading,
        foldersCard,
        volumeCard,
        actionCard,
        supportCard,
    )

    outer := container.New(layout.NewPaddedLayout(), content)
    return outer
}

func updateStartButton(label1, label2, folderOutputLabel *widget.Label, button *widget.Button) {
    if label1.Text != "No folder selected" && label2.Text != "No folder selected" && folderOutputLabel.Text != "No folder selected" {
        button.Enable()
    } else {
        button.Disable()
    }
}
