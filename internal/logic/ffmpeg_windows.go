//go:build windows
// +build windows

package logic

import (
  "archive/zip"
  "fmt"
  "io"
  "net/http"
  "os"
  "path/filepath"
  "strings"
)

var tempDir string

func getFFProbePath() (string, error) {
  getFFmpegPath()
  ffprobePath := filepath.Join(tempDir, "ffmpeg", "ffprobe.exe")
  if _, err := os.Stat(ffprobePath); os.IsNotExist(err) {
    return "", err
  }

  return ffprobePath, nil
}


func getFFmpegPath() (string, error) {
    if tmpDir == "" {
      tempDir = os.TempDir()
    }
    ffmpegDir := filepath.Join(tempDir, "ffmpeg")
    ffmpegPath := filepath.Join(ffmpegDir, "ffmpeg.exe")

    if _, err := os.Stat(ffmpegPath); os.IsNotExist(err) {
        zipPath := filepath.Join(tempDir, "ffmpeg.zip")
        ffmpegURL := "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-win64-gpl.zip"

        // Download ZIP file
        err := downloadFFmpeg(ffmpegURL, zipPath)
        if err != nil {
            return "", fmt.Errorf("failed to download FFmpeg: %v", err)
        }

        // Extract ZIP file
        err = unzip(zipPath, ffmpegDir)
        if err != nil {
            return "", fmt.Errorf("failed to extract FFmpeg: %v", err)
        }

        // Clean up ZIP file
        os.Remove(zipPath)

        // Find ffmpeg.exe in extracted files
        err = filepath.Walk(ffmpegDir, func(path string, info os.FileInfo, err error) error {
            if err != nil {
                return err
            }
            if info.Name() == "ffmpeg.exe" {
                ffmpegPath = path
                return io.EOF // Stop walking
            }
            return nil
        })
        if err != nil && err != io.EOF {
            return "", fmt.Errorf("failed to find ffmpeg.exe: %v", err)
        }
        if ffmpegPath == "" {
            return "", fmt.Errorf("ffmpeg.exe not found in extracted files")
        }
    }

    return ffmpegPath, nil
}

func downloadFFmpeg(url, destPath string) error {
    resp, err := http.Get(url)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    out, err := os.Create(destPath)
    if err != nil {
        return err
    }
    defer out.Close()

    _, err = io.Copy(out, resp.Body)
    return err
}

func unzip(src, dest string) error {
    r, err := zip.OpenReader(src)
    if err != nil {
        return err
    }
    defer r.Close()

    for _, f := range r.File {
        fpath := filepath.Join(dest, f.Name)

        if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
            return fmt.Errorf("invalid file path: %s", fpath)
        }

        if f.FileInfo().IsDir() {
            os.MkdirAll(fpath, os.ModePerm)
            continue
        }

        if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
            return err
        }

        outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
        if err != nil {
            return err
        }

        rc, err := f.Open()
        if err != nil {
            outFile.Close()
            return err
        }

        _, err = io.Copy(outFile, rc)
        outFile.Close()
        rc.Close()

        if err != nil {
            return err
        }
    }
    return nil
}

func cleanupTemp() {
    err := os.RemoveAll(tempDir)
    if err != nil {
        fmt.Printf("Error cleaning up temporary directory: %v\n", err)
    } else {
        fmt.Println("Temporary directory cleaned up successfully")
    }
}
