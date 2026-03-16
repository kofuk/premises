package main

import (
	"encoding/json"
	"os"
)

type ResourceType string

const (
	ResourceTypeGitHubReleases ResourceType = "github-releases"
	ResourceTypeLocal          ResourceType = "local"
)

type ResourceItem struct {
	Type            ResourceType `json:"type"`
	Destination     string       `json:"destination"`
	Source          string       `json:"source"`
	Checksum        string       `json:"checksum"`
	ApprovedLicense string       `json:"approvedLicense"`
}

func loadConfig(filename string) ([]ResourceItem, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var resources []ResourceItem
	if err := json.Unmarshal(content, &resources); err != nil {
		return nil, err
	}

	return resources, nil
}
