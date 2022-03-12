package mcversions

import (
	"errors"
	"net/http"

	"github.com/PuerkitoBio/goquery"
)

const (
	mcVersionsUrl = "https://mcversions.net/"
)

var (
	ErrHttpFailure = errors.New("Failed to retrive versions")
	ErrNotFound    = errors.New("Specified version not found")
)

type MCVersion struct {
	Name        string `json:"name"`
	IsStable    bool   `json:"isStable"`
	Channel     string `json:"channel"`
	ReleaseDate string `json:"releaseDate"`
}

func GetVersions() ([]MCVersion, error) {
	resp, err := http.Get(mcVersionsUrl)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, ErrHttpFailure
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var result []MCVersion

	doc.Find("h5").Each(func(_ int, s *goquery.Selection) {
		var channel string
		if text := s.Text(); text == "Stable Releases" {
			channel = "stable"
		} else if text == "Snapshot Preview" {
			channel = "snapshot"
		} else if text == "Beta" {
			channel = "beta"
		} else if text == "Alpha" {
			channel = "alpha"
		}
		isStable := channel == "stable"

		s.Siblings().Children().Each(func(_ int, s *goquery.Selection) {
			name, ok := s.Attr("id")
			if !ok {
				return
			}

			releaseDate := s.Find("time").Text()

			result = append(result, MCVersion{
				Name:        name,
				IsStable:    isStable,
				Channel:     channel,
				ReleaseDate: releaseDate,
			})
		})
	})

	return result, nil
}

func GetDownloadUrl(version string) (url string, err error) {
	resp, err := http.Get(mcVersionsUrl + "download/" + version)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", ErrHttpFailure
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	defer func() {
		recover()
	}()
	doc.Find("a[download]").Each(func(_ int, s *goquery.Selection) {
		if s.Text() == "Download Server Jar" {
			dlUrl, ok := s.Attr("href")
			if !ok {
				return
			}
			url = dlUrl
			// Can we do non-local-return without panic'ing here?
			panic(nil)
		}
	})

	if url == "" {
		return "", ErrNotFound
	}
	return
}
