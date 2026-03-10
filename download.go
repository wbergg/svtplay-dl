package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
)

type VideoResponse struct {
	VideoReferences []VideoReference `json:"videoReferences"`
}

type VideoReference struct {
	URL    string `json:"url"`
	Format string `json:"format"`
}

func DownloadEpisode(ep Episode, showSlug string) error {
	dir := filepath.Join(showSlug, ep.SeasonDir)
	filename := fmt.Sprintf("%d - %s.mp4", ep.EpisodeNumber, sanitizeFilename(ep.Title))
	outputPath := filepath.Join(dir, filename)

	if _, err := os.Stat(outputPath); err == nil {
		fmt.Printf("  Skipping (already exists): %s\n", outputPath)
		return nil
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	streamURL, err := fetchStreamURL(ep.VideoSvtId)
	if err != nil {
		return fmt.Errorf("fetching stream URL: %w", err)
	}

	if err := runFFmpeg(streamURL, outputPath); err != nil {
		return fmt.Errorf("downloading with ffmpeg: %w", err)
	}

	return nil
}

func fetchStreamURL(videoSvtId string) (string, error) {
	url := fmt.Sprintf("https://api.svt.se/video/%s", videoSvtId)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetching video API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("video API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading video API response: %w", err)
	}

	var videoResp VideoResponse
	if err := json.Unmarshal(body, &videoResp); err != nil {
		return "", fmt.Errorf("parsing video API response: %w", err)
	}

	for _, ref := range videoResp.VideoReferences {
		if ref.Format == "hls" || strings.Contains(ref.URL, ".m3u8") {
			return ref.URL, nil
		}
	}

	return "", fmt.Errorf("no HLS stream found for video %s", videoSvtId)
}

func runFFmpeg(streamURL, outputPath string) error {
	var stderr bytes.Buffer

	cmd := exec.Command("ffmpeg",
		"-i", streamURL,
		"-c", "copy",
		"-bsf:a", "aac_adtstoasc",
		"-n",
		outputPath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Get last few lines of ffmpeg output for the error message
		lines := strings.Split(strings.TrimSpace(stderr.String()), "\n")
		tail := lines
		if len(tail) > 5 {
			tail = tail[len(tail)-5:]
		}
		return fmt.Errorf("%w\nffmpeg output:\n%s", err, strings.Join(tail, "\n"))
	}
	return nil
}

func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "",
		"?", "",
		"\"", "",
		"<", "",
		">", "",
		"|", "",
	)
	return replacer.Replace(name)
}
