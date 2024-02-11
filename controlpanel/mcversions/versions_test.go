package mcversions

import (
	"net/http"
	"testing"

	"context"
	"errors"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/kofuk/premises/controlpanel/caching"
	"github.com/stretchr/testify/assert"
)

var testingLauncherMeta = launcherMeta{
	Versions: []VersionInfo{
		{
			ID:          "1.20.1",
			Type:        "release",
			URL:         "https://launchermeta/version0.json",
			ReleaseTime: "2023-11-01T12:30:52+00:00",
		},
		{
			ID:          "1.20.0",
			Type:        "release",
			URL:         "https://launchermeta/version1.json",
			ReleaseTime: "2023-10-31T12:30:52+00:00",
		},
		{
			ID:          "23w43b",
			Type:        "snapshot",
			URL:         "https://launchermeta/version2.json",
			ReleaseTime: "2023-10-30T12:30:52+00:00",
		},
	},
}

var testingVersionMeta0 = `{"downloads":{"server":{"url":"https://launchermeta/version0.jar"}}}`
var testingVersionMeta1 = `{"downloads":{"server":{"url":"https://launchermeta/version1.jar"}}}`
var testingVersionMeta2 = `{"downloads":{"server":{"url":"https://launchermeta/version2.jar"}}}`

type MapCacheImpl struct {
	entries map[string][]byte
}

func (self *MapCacheImpl) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	self.entries[key] = value
	return nil
}

func (self *MapCacheImpl) Get(ctx context.Context, key string) ([]byte, error) {
	val, ok := self.entries[key]
	if !ok {
		return nil, errors.New("")
	}
	return val, nil
}

func (self *MapCacheImpl) Del(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		delete(self.entries, key)
	}
	return nil
}

func setupResponders() {
	httpmock.RegisterResponder(http.MethodGet, "https://launchermeta/version_manifest.json", httpmock.NewJsonResponderOrPanic(http.StatusOK, testingLauncherMeta))
	httpmock.RegisterResponder(http.MethodGet, "https://launchermeta/version0.json", httpmock.NewStringResponder(http.StatusOK, testingVersionMeta0))
	httpmock.RegisterResponder(http.MethodGet, "https://launchermeta/version1.json", httpmock.NewStringResponder(http.StatusOK, testingVersionMeta1))
	httpmock.RegisterResponder(http.MethodGet, "https://launchermeta/version2.json", httpmock.NewStringResponder(http.StatusOK, testingVersionMeta2))
}

func Test_GetVersions(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	setupResponders()

	provider := New(caching.New(&MapCacheImpl{
		entries: make(map[string][]byte),
	}), ManifestURL("https://launchermeta/version_manifest.json"))
	versions1, err := provider.GetVersions(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, testingLauncherMeta.Versions, versions1)

	versions2, err := provider.GetVersions(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, versions1, versions2)

	assert.Equal(t, 1, httpmock.GetCallCountInfo()["GET https://launchermeta/version_manifest.json"])
	assert.Equal(t, 3, len(versions1))
}

func Test_GetDownloadURL(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	setupResponders()

	provider := New(caching.New(&MapCacheImpl{
		entries: make(map[string][]byte),
	}), ManifestURL("https://launchermeta/version_manifest.json"))

	url1, err := provider.GetDownloadURL(context.Background(), "1.20.1")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "https://launchermeta/version0.jar", url1)

	url2, err := provider.GetDownloadURL(context.Background(), "1.20.1")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "https://launchermeta/version0.jar", url2)

	assert.Equal(t, 1, httpmock.GetCallCountInfo()["GET https://launchermeta/version_manifest.json"])
	assert.Equal(t, 1, httpmock.GetCallCountInfo()["GET https://launchermeta/version0.json"])

	url3, err := provider.GetDownloadURL(context.Background(), "1.20.0")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "https://launchermeta/version1.jar", url3)

	assert.Equal(t, 1, httpmock.GetCallCountInfo()["GET https://launchermeta/version1.json"])
}
