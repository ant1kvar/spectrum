<div align="center">

# Spectrum

**Terminal-based audio waveform visualizer for internet radio streams**

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Linux%20|%20macOS%20|%20RPi-lightgrey)]()

<img width="582" height="260" alt="spectrum screenshot" src="https://github.com/user-attachments/assets/b678d638-d284-4cc0-9241-c90a145d11ac" />

![demo](https://github.com/user-attachments/assets/6532f4ef-b5f1-421a-b7e5-55588ca17a41)

</div>

---

## Features

- Real-time waveform visualization
- Stream metadata extraction (artist, track)
- Configurable appearance (size, characters, colors)
- Low latency (~30 FPS)
- Optimized for Raspberry Pi

---

## Requirements

| Dependency | Purpose |
|------------|---------|
| Go 1.22+   | Build   |
| ffmpeg / ffprobe | Audio processing |
| ffplay     | Audio playback |

```bash
# Debian / Ubuntu / Raspberry Pi
sudo apt install ffmpeg
```

---

## Installation

```bash
go get github.com/ant1kvar/spectrum
```

---

## Quick Start

```bash
cd example && go run .
```

---

## Usage

### Basic Example

```go
package main

import (
    "context"
    "fmt"
    "github.com/ant1kvar/spectrum"
)

func main() {
    cfg := spectrum.DefaultConfig()
    cfg.Width = 60
    cfg.Height = 12
    cfg.Char = "|"

    vis := spectrum.New(cfg)

    spectrum.ClearScreen()

    go func() {
        track := vis.FetchTrack()
        fmt.Printf("Now playing: %s - %s\n", track.Artist, track.Title)
    }()

    vis.StartFromURL(context.Background(), "http://stream-url")
}
```

### With Track Info

```go
track := vis.FetchTrack()
fmt.Printf("Now playing: %s - %s\n", track.Artist, track.Title)
```

---

## Configuration

```go
cfg := spectrum.DefaultConfig()
cfg.Width = 80
cfg.Height = 16
cfg.Char = "|"
cfg.SmoothFactor = 0.9
```

| Option | Default | Description |
|:-------|:-------:|:------------|
| `Width` | 60 | Display width (characters) |
| `Height` | 12 | Display height (rows) |
| `SampleRate` | 44100 | Audio sample rate (Hz) |
| `ChunkSize` | 1024 | Samples per buffer |
| `FPS` | 30 | Frames per second |
| `SmoothFactor` | 0.9 | Smoothing (0-1) |
| `Char` | `\|` | Bar character |
| `BarSpacing` | 1 | Gap between bars |
| `Amplify` | 2.5 | Amplitude multiplier |
| `ShowStatus` | true | Show status line |

---

## API Reference

### Visualizer

```go
// Create
vis := spectrum.New(cfg)

// Start from URL
vis.StartFromURL(ctx, "http://stream-url")

// Start from io.Reader (PCM s16le mono)
vis.StartFromReader(ctx, reader)

// Control
vis.Stop()
vis.IsRunning()
```

### Track Metadata

```go
// Fetch via ffprobe
track := vis.FetchTrack()
track.Artist  // Artist name
track.Title   // Track title
track.Raw     // Raw metadata

// Get cached (no request)
track := vis.GetTrack()
```

### Data Access

```go
vis.GetWaveform()  // []float64 - current values
vis.Render()       // string - rendered frame
```

### Utilities

```go
spectrum.ClearScreen()
spectrum.ShowCursor()
```

---

## Project Structure

```
spectrum/
├── spectrum.go      # Library
├── example/
│   ├── main.go      # Example app
│   └── go.mod
├── go.mod
└── README.md
```

---

## License

MIT
