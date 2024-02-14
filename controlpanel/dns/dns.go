package dns

import (
	"context"
	"net"
)

type DNSProvider interface {
	UpdateV4(ctx context.Context, domain string, addr net.IP) error
}

type DNSService struct {
	d          DNSProvider
	domainName string
}

func New(d DNSProvider, domainName string) *DNSService {
	return &DNSService{
		d:          d,
		domainName: domainName,
	}
}

func (self DNSService) UpdateV4(ctx context.Context, addr net.IP) error {
	return self.d.UpdateV4(ctx, self.domainName, addr.To4())
}
