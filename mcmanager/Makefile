GO = go

.PHONY: all
all:
	CGO_ENABLED=0 $(GO) build -o premises-mcmanager -ldflags \
	    '-X github.com/kofuk/premises-mcmanager/metadata.Revision=$(shell [ -e ../.git ]&&git rev-parse --short HEAD||echo unkonwn)'

.PHONY: clean
clean:
	$(RM) premises-mcmanager
