package dns

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/cloudflare/cloudflare-go"
)

type CloudflareDNS struct {
	api            *cloudflare.API
	zoneIdentifier *cloudflare.ResourceContainer
}

func NewCloudflareDNS(token, zoneID string, options ...cloudflare.Option) (*CloudflareDNS, error) {
	api, err := cloudflare.NewWithAPIToken(token, options...)
	if err != nil {
		return nil, err
	}

	return &CloudflareDNS{
		api:            api,
		zoneIdentifier: cloudflare.ZoneIdentifier(zoneID),
	}, nil
}

func (self *CloudflareDNS) update(ctx context.Context, domainName string, addr net.IP) error {
	if len(addr) != net.IPv4len {
		return errors.New("IPv6 not supported")
	}

	records, _, err := self.api.ListDNSRecords(ctx, self.zoneIdentifier, cloudflare.ListDNSRecordsParams{
		Type:    "A",
		Name:    domainName,
		Proxied: cloudflare.BoolPtr(false),
		Match:   "all",
	})
	if err != nil {
		return err
	}

	if len(records) == 0 {
		return fmt.Errorf("Record does not exist")
	}

	recordID := records[0].ID
	if _, err := self.api.UpdateDNSRecord(ctx, self.zoneIdentifier, cloudflare.UpdateDNSRecordParams{
		Content: addr.String(),
		ID:      recordID,
	}); err != nil {
		return err
	}

	return nil
}

func (self *CloudflareDNS) UpdateV4(ctx context.Context, domainName string, addr net.IP) error {
	return self.update(ctx, domainName, addr)
}
