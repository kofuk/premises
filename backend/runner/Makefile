GO = go
DOCKER = docker

.PHONY: all
all:

.PHONY: sh
sh:
	$(DOCKER) container exec -it $(shell $(DOCKER) container ls -aqf label=org.kofuk.premises.managed) bash

.PHONY: res
resources/cmd/resources.tar.zst: resources/resources.json
	cd resources/cmd && go run . -input ../resources.json

.PHONY: deploy-dev
deploy-dev: resources/cmd/resources.tar.zst
	mkdir -p /tmp/premises
	CGO_ENABLED=0 GOOS=linux $(GO) build -o /tmp/premises/premises-runner
	cp debug_env.sh /tmp/premises/env
	cp resources/cmd/resources.tar.zst /tmp/premises/resources.tar.zst
