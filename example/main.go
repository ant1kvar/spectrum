package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/ant1kvar/spectrum"
)

func main() {
	streamURL := "http://s2-webradio.rockantenne.de/rockantenne"

	cfg := spectrum.DefaultConfig()
	cfg.Width = 60
	cfg.Height = 12
	cfg.Char = "|"
	cfg.SmoothFactor = 0.9

	vis := spectrum.New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
		spectrum.ShowCursor()
		os.Exit(0)
	}()

	go playAudio(ctx, streamURL)
	go trackUpdater(ctx, vis)

	spectrum.ClearScreen()

	if err := vis.StartFromURL(ctx, streamURL); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}

func playAudio(ctx context.Context, streamURL string) {
	cmd := exec.CommandContext(ctx, "ffplay", "-nodisp", "-autoexit", streamURL)
	cmd.Run()
}

func trackUpdater(ctx context.Context, vis *spectrum.Visualizer) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			track := vis.FetchTrack()
			if track.Raw != "" {
				fmt.Printf("\033[1;0H\033[KNow playing: %s\n", track.Raw)
			} else {
				fmt.Print("\033[1;0H\033[K")
			}
		}
	}
}
