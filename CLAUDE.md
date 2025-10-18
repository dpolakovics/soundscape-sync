# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Soundscape Sync is a Go GUI application built with Fyne that synchronizes soundscape audio files with audiobook audio files. It uses FFmpeg to combine audio from two folders (soundscape + audiobook) into merged output files with configurable volume mixing.

**Key Purpose**: Users provide a soundscape folder, an audiobook folder, and an output folder. The app processes matching files in order, combining them using FFmpeg filter complex commands while preserving metadata from the audiobook.

## Development Commands

### Building
- **Standard build**: `go build`
- **Cross-compile for Linux**: `fyne-cross linux`
- **Cross-compile for Windows**: `fyne-cross windows`
- **macOS Universal Binary**:
  ```bash
  GOOS=darwin GOARCH=arm64 go build -o SoundscapeSync_arm64
  CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o SoundscapeSync_amd64
  lipo -create -output SoundscapeSync SoundscapeSync_amd64 SoundscapeSync_arm64
  ```

### Running
- **Run locally**: `go run main.go`
- **Linux requirement**: ffmpeg must be installed on Linux systems

### Dependencies
- **Install fyne-cross** (for cross-compilation): `go install github.com/fyne-io/fyne-cross@latest`
- Go version: 1.22+

## Architecture

### High-Level Structure

The codebase follows a clean separation between UI, business logic, and platform-specific code:

```
main.go              -> Entry point, version checking, update dialog
internal/ui/         -> Fyne GUI components and folder selection
internal/logic/      -> Audio processing and FFmpeg orchestration
internal/ffmpeg/     -> Platform-specific FFmpeg binary paths
```

### Core Processing Flow

1. **UI Layer** (`internal/ui/windows.go`):
   - User selects soundscape folder, audiobook folder, and output folder
   - Configures mix preset (volume balance between audiobook and soundscape)
   - Initiates sync via `logic.CombineFiles()`

2. **Logic Layer** (`internal/logic/sync.go`):
   - Validates matching file counts between folders
   - For each file pair:
     - Detects channel count (2/6/10/12 for stereo/5.1/7.1.2/7.1.4)
     - Builds FFmpeg filter_complex arguments based on channel configuration
     - Executes FFmpeg command with progress tracking
     - Copies metadata and cover art from audiobook to output

3. **Channel-Specific Logic**:
   - `sync-stereo.go`: Handles 2-channel stereo mixing
   - `sync-5.1.go`: Handles 6-channel surround sound
   - `sync-7.1.2.go`: Handles 10-channel Dolby Atmos
   - `sync-7.1.4.go`: Handles 12-channel Dolby Atmos

### Platform-Specific Code

- **FFmpeg Binaries**: Each platform (darwin_arm64, darwin_amd64, linux_amd64, windows_amd64) has its own build-tagged file in `internal/ffmpeg/` that imports static FFmpeg binaries
- **System Process Attributes**: `sysproc_windows.go` and `sysproc_default.go` handle platform-specific exec configuration
- **Native Folder Dialogs**: `internal/ui/windows.go` attempts native OS folder pickers (PowerShell for Windows, osascript for macOS, zenity/kdialog for Linux) before falling back to Fyne's dialog

### FFmpeg Filter Complex Strategy

The app uses FFmpeg's `filter_complex` to:
1. Apply volume adjustments to audiobook (input 0) and soundscape (input 1)
2. Pad the shorter audio stream with silence
3. Merge streams and map channels appropriately for output format
4. Map metadata from the audiobook file (input 0) to preserve title, artist, etc.
5. Copy cover art if present in either input file

Each channel configuration (stereo, 5.1, 7.1.2, 7.1.4) has custom pan/amerge logic in its respective sync-*.go file.

## Version Management

- **Version location**: Update `currentTag` in `main.go:24` (e.g., "v1.1.1")
- **Build metadata**: Update `FyneApp.toml` with Version and Build number
- The app automatically checks for updates on startup using the GitHub API

## Release Process

- **Trigger**: Push a tag matching `v*.*` (e.g., `v1.2.0`)
- **Workflow**: `.github/workflows/release.yml` builds for Linux, Windows, and macOS
- **macOS Signing**: Uses secrets for code signing and notarization (MAC_CERT_P12, APPLE_ID, etc.)
- **Output**: Creates GitHub release with binaries for all platforms

## Key Files

- `main.go`: Application entry point with update checking
- `internal/ui/windows.go`: Complete GUI definition and folder selection
- `internal/logic/sync.go`: Core file combination logic, FFmpeg execution
- `internal/logic/sync-{stereo,5.1,7.1.2,7.1.4}.go`: Channel-specific FFmpeg filter arguments
- `FyneApp.toml`: App metadata (name, bundle ID, version, build number)

## Important Constraints

- **File matching**: Both input folders must contain the same number of files, sorted lexicographically
- **Supported formats**: .mp3, .wav, .m4b, .mp4
- **Supported channel counts**: 2 (stereo), 6 (5.1), 10 (7.1.2), 12 (7.1.4)
- **FFmpeg requirement**: Linux users must have ffmpeg installed; other platforms use embedded static binaries
