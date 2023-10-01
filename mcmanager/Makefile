GO = go

.PHONY: all
all:
	CGO_ENABLED=0 $(GO) build -o premises-mcmanager -ldflags \
	    '-X github.com/kofuk/premises/mcmanager/metadata.Revision=$(shell [ -e ../.git ]&&git rev-parse --short HEAD||echo unkonwn)'

.PHONY: deploy-dev
deploy-dev:
	mkdir -p /tmp/premises
	CGO_ENABLED=0 GOOS=linux $(GO) build -o /tmp/premises/premises-mcmanager

.PHONY: clean
clean:
	$(RM) premises-mcmanager

.PHONY: setup-gamedatafs
setup-gamedatafs:
	mkdir -p /tmp/premises/gamedata
	fallocate -l 1GiB /tmp/premises/gamedata.img
	mkfs.btrfs -f /tmp/premises/gamedata.img
	sudo mount /tmp/premises/gamedata.img /tmp/premises/gamedata
	sudo chown -R '1000:1000' /tmp/premises/gamedata

.PHONY: run-debug-privilaged-proc
run-debug-privilaged-proc:
	tmpname="/tmp/$$(uuidgen)";                                                 \
	$(GO) build -o "$${tmpname}";                                               \
	(                                                                           \
		sleep 30;                                                               \
		$(RM) "$${tmpname}"                                                     \
	) &                                                                         \
	pid=$$!;                                                                    \
	sudo -E PREMISES_RUNNER_DEBUG=true "$${tmpname}" --privileged-helper;       \
	wait "$${pid}"
