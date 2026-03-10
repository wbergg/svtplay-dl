package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	seasonFlag := flag.Int("season", 0, "Download only this season number (default: all)")
	listFlag := flag.Bool("list", false, "List available seasons and episodes without downloading")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <SVTPlay show URL>\n\nFlags:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s https://www.svtplay.se/bolibompa-draken-foljer-med\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -list https://www.svtplay.se/bolibompa-draken-foljer-med\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -season 3 https://www.svtplay.se/bolibompa-draken-foljer-med\n", os.Args[0])
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	url := flag.Arg(0)
	if !strings.Contains(url, "svtplay.se/") {
		log.Fatalf("Invalid URL: must be a svtplay.se show URL")
	}

	// Extract show slug from URL
	parts := strings.Split(strings.TrimRight(url, "/"), "/")
	showSlug := parts[len(parts)-1]

	fmt.Printf("Fetching show data from %s...\n", url)
	episodes, showName, err := FetchAndParseShow(url)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("Found show: %s (%d episodes)\n\n", showName, len(episodes))

	// Filter by season if requested
	if *seasonFlag > 0 {
		var filtered []Episode
		for _, ep := range episodes {
			if ep.SeasonNumber == *seasonFlag {
				filtered = append(filtered, ep)
			}
		}
		if len(filtered) == 0 {
			log.Fatalf("No episodes found for season %d", *seasonFlag)
		}
		episodes = filtered
	}

	// List mode
	if *listFlag {
		currentSeason := 0
		for _, ep := range episodes {
			if ep.SeasonNumber != currentSeason {
				currentSeason = ep.SeasonNumber
				fmt.Printf("Season %d:\n", currentSeason)
			}
			fmt.Printf("  %d - %s\n", ep.EpisodeNumber, ep.Title)
		}
		return
	}

	// Download mode
	type failure struct {
		Episode Episode
		Err     error
	}

	success := 0
	var failures []failure
	for _, ep := range episodes {
		fmt.Printf("Downloading [%s] %d - %s\n", ep.SeasonDir, ep.EpisodeNumber, ep.Title)
		if err := DownloadEpisode(ep, showSlug); err != nil {
			log.Printf("  Failed: %v", err)
			failures = append(failures, failure{Episode: ep, Err: err})
		} else {
			success++
		}
	}

	fmt.Printf("\nDone! %d downloaded, %d failed.\n", success, len(failures))

	if len(failures) > 0 {
		fmt.Printf("\nFailed episodes:\n")
		for _, f := range failures {
			fmt.Printf("  [%s] %d - %s: %v\n", f.Episode.SeasonDir, f.Episode.EpisodeNumber, f.Episode.Title, f.Err)
		}
	}
}
