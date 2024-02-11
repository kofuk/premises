package dns

import (
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/cloudflare/cloudflare-go"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func Test_CloudflareDNS_UpdateV4(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder(http.MethodGet, "http://cf/client/v4/zones/yyy/dns_records?match=all&name=game.example.com&page=1&per_page=100&proxied=false&type=A",
		httpmock.NewJsonResponderOrPanic(http.StatusOK, map[string]any{
			"errors":   make([]any, 0),
			"messages": make([]any, 0),
			"result": []map[string]any{
				{
					"content":     "192.0.2.1",
					"name":        "game.example.com",
					"proxied":     false,
					"type":        "A",
					"comment":     "",
					"created_on":  "2014-01-01T05:20:00.12345Z",
					"id":          "record1",
					"modified_on": "2014-01-01T05:20:00.12345Z",
					"proxyable":   true,
					"ttl":         60,
					"zone_id":     "yyy",
					"zone_name":   "example.com",
				},
			},
		}))
	httpmock.RegisterResponder(http.MethodPatch, "http://cf/client/v4/zones/yyy/dns_records/record1",
		httpmock.NewJsonResponderOrPanic(http.StatusOK, map[string]any{
			"errors":   make([]any, 0),
			"messages": make([]any, 0),
			"result": map[string]any{
				"content":     "192.0.2.2",
				"name":        "game.example.com",
				"proxied":     false,
				"type":        "A",
				"comment":     "",
				"created_on":  "2014-01-01T05:20:00.12345Z",
				"id":          "record1",
				"modified_on": "2014-01-01T05:20:00.12345Z",
				"proxyable":   true,
				"ttl":         60,
				"zone_id":     "yyy",
				"zone_name":   "example.com",
			},
		}))

	cfdns, err := NewCloudflareDNS("xxx", "yyy", cloudflare.BaseURL("http://cf/client/v4"), cloudflare.Debug(true))
	dnsProvider := New(cfdns, "game.example.com")

	err = dnsProvider.UpdateV4(context.Background(), net.ParseIP("192.0.2.2"))
	assert.NoError(t, err)

	assert.Equal(t, 1, httpmock.GetCallCountInfo()["GET http://cf/client/v4/zones/yyy/dns_records?match=all&name=game.example.com&page=1&per_page=100&proxied=false&type=A"])
	assert.Equal(t, 1, httpmock.GetCallCountInfo()["PATCH http://cf/client/v4/zones/yyy/dns_records/record1"])
}
