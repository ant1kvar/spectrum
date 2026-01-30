package spectrum

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	Width        int
	Height       int
	SampleRate   int
	ChunkSize    int
	FPS          int
	SmoothFactor float64
	Char         string
	BarSpacing   int
	Amplify      float64
	ShowStatus   bool
}

func DefaultConfig() Config {
	return Config{
		Width:        60,
		Height:       12,
		SampleRate:   44100,
		ChunkSize:    1024,
		FPS:          30,
		SmoothFactor: 0.9,
		Char:         "|",
		BarSpacing:   1,
		Amplify:      2.5,
		ShowStatus:   true,
	}
}

type TrackInfo struct {
	Title  string
	Artist string
	Raw    string
}

type Visualizer struct {
	config    Config
	waveform  []float64
	smoothed  []float64
	track     TrackInfo
	streamURL string
	mu        sync.RWMutex
	cancel    context.CancelFunc
	running   bool
}

func New(cfg Config) *Visualizer {
	if cfg.Width == 0 {
		cfg.Width = 60
	}
	if cfg.Height == 0 {
		cfg.Height = 12
	}
	if cfg.SampleRate == 0 {
		cfg.SampleRate = 44100
	}
	if cfg.ChunkSize == 0 {
		cfg.ChunkSize = 1024
	}
	if cfg.FPS == 0 {
		cfg.FPS = 30
	}
	if cfg.Char == "" {
		cfg.Char = "|"
	}
	if cfg.BarSpacing == 0 {
		cfg.BarSpacing = 1
	}
	if cfg.Amplify == 0 {
		cfg.Amplify = 2.5
	}

	return &Visualizer{
		config:   cfg,
		waveform: make([]float64, cfg.Width),
		smoothed: make([]float64, cfg.Width),
	}
}

func (v *Visualizer) StartFromURL(ctx context.Context, streamURL string) error {
	ctx, v.cancel = context.WithCancel(ctx)
	v.running = true
	v.streamURL = streamURL

	visCmd := exec.CommandContext(ctx, "ffmpeg",
		"-probesize", "32k",
		"-analyzeduration", "0",
		"-fflags", "nobuffer",
		"-flags", "low_delay",
		"-i", streamURL,
		"-ac", "1",
		"-ar", strconv.Itoa(v.config.SampleRate),
		"-f", "s16le",
		"-acodec", "pcm_s16le",
		"-vn",
		"-",
	)

	stdout, err := visCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create pipe: %w", err)
	}

	if err := visCmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	go func() {
		<-ctx.Done()
		visCmd.Process.Kill()
	}()

	reader := bufio.NewReaderSize(stdout, v.config.ChunkSize*4)
	return v.processStream(ctx, reader)
}

func (v *Visualizer) StartFromReader(ctx context.Context, reader io.Reader) error {
	ctx, v.cancel = context.WithCancel(ctx)
	v.running = true

	bufReader := bufio.NewReaderSize(reader, v.config.ChunkSize*4)
	return v.processStream(ctx, bufReader)
}

func (v *Visualizer) Stop() {
	if v.cancel != nil {
		v.cancel()
	}
	v.running = false
}

func (v *Visualizer) IsRunning() bool {
	return v.running
}

func (v *Visualizer) GetWaveform() []float64 {
	v.mu.RLock()
	defer v.mu.RUnlock()
	result := make([]float64, len(v.smoothed))
	copy(result, v.smoothed)
	return result
}

func (v *Visualizer) Render() string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.renderFrame(v.smoothed)
}

func (v *Visualizer) GetTrack() TrackInfo {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.track
}

func (v *Visualizer) FetchTrack() TrackInfo {
	if v.streamURL == "" {
		return v.track
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		v.streamURL,
	)

	output, err := cmd.Output()
	if err != nil {
		return v.track
	}

	var result struct {
		Format struct {
			Tags map[string]string `json:"tags"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return v.track
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	if title, ok := result.Format.Tags["StreamTitle"]; ok {
		v.track.Raw = title
		if parts := strings.SplitN(title, " - ", 2); len(parts) == 2 {
			v.track.Artist = strings.TrimSpace(parts[0])
			v.track.Title = strings.TrimSpace(parts[1])
		} else {
			v.track.Title = title
			v.track.Artist = ""
		}
	} else if title, ok := result.Format.Tags["icy-name"]; ok {
		v.track.Raw = title
		v.track.Title = title
	}

	return v.track
}

func (v *Visualizer) processStream(ctx context.Context, reader *bufio.Reader) error {
	rawBuffer := make([]byte, v.config.ChunkSize*2)
	buffer := make([]int16, v.config.ChunkSize)
	waveform := make([]float64, v.config.Width)

	updateInterval := time.Second / time.Duration(v.config.FPS)

	for {
		select {
		case <-ctx.Done():
			v.running = false
			return ctx.Err()
		default:
		}

		startTime := time.Now()

		n, err := io.ReadFull(reader, rawBuffer)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				continue
			}
			return err
		}
		if n < len(rawBuffer) {
			continue
		}

		for i := range v.config.ChunkSize {
			buffer[i] = int16(rawBuffer[i*2]) | int16(rawBuffer[i*2+1])<<8
		}

		v.convertToWaveform(buffer, waveform)

		v.mu.Lock()
		for i := range waveform {
			v.smoothed[i] = v.smoothed[i]*(1-v.config.SmoothFactor) + waveform[i]*v.config.SmoothFactor
		}
		v.mu.Unlock()

		fmt.Print(v.Render())

		elapsed := time.Since(startTime)
		if elapsed < updateInterval {
			time.Sleep(updateInterval - elapsed)
		}
	}
}

func (v *Visualizer) convertToWaveform(buffer []int16, waveform []float64) {
	samplesPerColumn := len(buffer) / len(waveform)

	for col := range waveform {
		start := col * samplesPerColumn
		end := min(start+samplesPerColumn, len(buffer))

		sum := 0.0
		for i := start; i < end; i++ {
			value := float64(buffer[i]) / 32768.0
			sum += value * value
		}

		if samplesPerColumn > 0 {
			waveform[col] = math.Sqrt(sum / float64(samplesPerColumn))
		}
	}
}

func (v *Visualizer) renderFrame(waveform []float64) string {
	var sb strings.Builder
	sb.Grow(v.config.Width * v.config.Height * 4)

	sb.WriteString("\033[2;0H")
	sb.WriteString("\033[?25l")

	midline := v.config.Height / 2

	for row := range v.config.Height {
		for col := range v.config.Width {
			if v.config.BarSpacing > 1 && col%v.config.BarSpacing != 0 {
				sb.WriteByte(' ')
				continue
			}

			waveIdx := col
			if v.config.BarSpacing > 1 {
				waveIdx = col / v.config.BarSpacing
			}
			if waveIdx >= len(waveform) {
				sb.WriteByte(' ')
				continue
			}

			height := int(waveform[waveIdx] * v.config.Amplify * float64(midline-1))
			height = min(height, midline-1)

			if row >= midline-height && row <= midline+height && height > 0 {
				sb.WriteString(v.config.Char)
			} else {
				sb.WriteByte(' ')
			}
		}
		sb.WriteByte('\n')
	}

	if v.config.ShowStatus {
		sb.WriteString(fmt.Sprintf("Audio Visualizer | %dHz | %d samples | %d FPS\n",
			v.config.SampleRate, v.config.ChunkSize, v.config.FPS))
	}

	return sb.String()
}

func ClearScreen() {
	fmt.Print("\033[2J")
}

func ShowCursor() {
	fmt.Print("\033[?25h")
}
