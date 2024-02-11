package dns

import (
	"context"
	"net"
)

type DNSProviderImpl interface {
	UpdateV4(ctx context.Context, domain string, addr net.IP) error
}

type DNSProvider struct {
	d          DNSProviderImpl
	domainName string
}

func New(d DNSProviderImpl, domainName string) *DNSProvider {
	return &DNSProvider{
		d:          d,
		domainName: domainName,
	}
}

func (self DNSProvider) UpdateV4(ctx context.Context, addr net.IP) error {
	return self.d.UpdateV4(ctx, self.domainName, addr.To4())
}
