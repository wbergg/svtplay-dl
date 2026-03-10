# svtplay-dl

A command-line tool to download TV shows from SVT Play, written in Go.

## Requirements

- Go 1.21+
- [ffmpeg](https://ffmpeg.org/) installed and available in `$PATH`

## Install

```sh
go install github.com/wbergg/svtplay-dl@latest
```

Or build from source:

```sh
go build -o svtplay-dl .
```

## Usage

```sh
# Download all episodes of a show
svtplay-dl https://www.svtplay.se/bolibompa-draken-foljer-med

# List available seasons and episodes
svtplay-dl -list https://www.svtplay.se/bolibompa-draken-foljer-med

# Download only a specific season
svtplay-dl -season 3 https://www.svtplay.se/bolibompa-draken-foljer-med
```

Episodes are saved as `.mp4` files organized by show and season:

```
show-slug/
  Season 1/
    1 - Episode Title.mp4
    2 - Episode Title.mp4
  Season 2/
    ...
```

For shows where seasons have names instead of numbers (e.g. "Fantus leker", "Fantus musikantus"), the season name is used as the directory:

```
fantus/
  Fantus leker/
    1 - Klappa.mp4
    2 - Runt och runt.mp4
  Fantus musikantus/
    1 - ...
```

Already downloaded episodes are automatically skipped.
