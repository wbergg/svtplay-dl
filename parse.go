package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type Episode struct {
	ShowName      string
	SeasonNumber  int
	SeasonDir     string
	EpisodeNumber int
	Title         string
	VideoSvtId    string
}

type NextData struct {
	Props json.RawMessage `json:"props"`
	// urqlState is at the top level in some versions
	UrqlState map[string]UrqlEntry `json:"urqlState"`
}

type UrqlEntry struct {
	Data string `json:"data"`
}

type DetailsPage struct {
	DetailsPageByPath DetailsByPath `json:"detailsPageByPath"`
}

type DetailsByPath struct {
	Item    ShowItem `json:"item"`
	Modules []Module `json:"modules"`
}

type ShowItem struct {
	Name string `json:"name"`
}

type Module struct {
	Selection Selection `json:"selection"`
}

type Selection struct {
	SelectionType string   `json:"selectionType"`
	Name          string   `json:"name"`
	Items         []Teaser `json:"items"`
}

type Teaser struct {
	Heading string     `json:"heading"`
	Item    TeaserItem `json:"item"`
}

type TeaserItem struct {
	VideoSvtId string `json:"videoSvtId"`
}

var seasonRegexp = regexp.MustCompile(`\d+`)

func FetchAndParseShow(url string) ([]Episode, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("fetching show page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("reading response body: %w", err)
	}

	html := string(body)

	// Extract __NEXT_DATA__ JSON
	const startTag = `<script id="__NEXT_DATA__" type="application/json">`
	const endTag = `</script>`

	startIdx := strings.Index(html, startTag)
	if startIdx == -1 {
		return nil, "", fmt.Errorf("could not find __NEXT_DATA__ script tag")
	}
	startIdx += len(startTag)

	endIdx := strings.Index(html[startIdx:], endTag)
	if endIdx == -1 {
		return nil, "", fmt.Errorf("could not find closing script tag")
	}

	jsonData := html[startIdx : startIdx+endIdx]

	var nextData NextData
	if err := json.Unmarshal([]byte(jsonData), &nextData); err != nil {
		return nil, "", fmt.Errorf("parsing __NEXT_DATA__: %w", err)
	}

	// If urqlState not at top level, try props.urqlState then props.pageProps.urqlState
	if len(nextData.UrqlState) == 0 {
		var props struct {
			UrqlState map[string]UrqlEntry `json:"urqlState"`
			PageProps struct {
				UrqlState map[string]UrqlEntry `json:"urqlState"`
			} `json:"pageProps"`
		}
		if err := json.Unmarshal(nextData.Props, &props); err == nil {
			if len(props.UrqlState) > 0 {
				nextData.UrqlState = props.UrqlState
			} else {
				nextData.UrqlState = props.PageProps.UrqlState
			}
		}
	}

	if len(nextData.UrqlState) == 0 {
		return nil, "", fmt.Errorf("no urqlState found in page data")
	}

	// Find the entry containing episode data
	var detailsPage DetailsPage
	found := false
	for _, entry := range nextData.UrqlState {
		var dp DetailsPage
		if err := json.Unmarshal([]byte(entry.Data), &dp); err != nil {
			continue
		}
		if len(dp.DetailsPageByPath.Modules) > 0 {
			detailsPage = dp
			found = true
			break
		}
	}

	if !found {
		return nil, "", fmt.Errorf("could not find details page data in urqlState")
	}

	showName := detailsPage.DetailsPageByPath.Item.Name

	var episodes []Episode

	for _, mod := range detailsPage.DetailsPageByPath.Modules {
		if mod.Selection.SelectionType != "season" {
			continue
		}

		seasonNum := 1
		seasonDir := ""
		matches := seasonRegexp.FindString(mod.Selection.Name)
		if matches != "" {
			if n, err := strconv.Atoi(matches); err == nil {
				seasonNum = n
			}
			seasonDir = fmt.Sprintf("Season %d", seasonNum)
		} else {
			seasonDir = mod.Selection.Name
		}

		for i, item := range mod.Selection.Items {
			if item.Item.VideoSvtId == "" {
				continue
			}

			episodes = append(episodes, Episode{
				ShowName:      showName,
				SeasonNumber:  seasonNum,
				SeasonDir:     seasonDir,
				EpisodeNumber: i + 1,
				Title:         item.Heading,
				VideoSvtId:    item.Item.VideoSvtId,
			})
		}
	}

	if len(episodes) == 0 {
		return nil, "", fmt.Errorf("no episodes found on page")
	}

	return episodes, showName, nil
}
