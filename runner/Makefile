GO = go
DOCKER = docker

.PHONY: all
all:

.PHONY: sh
sh:
	$(DOCKER) container exec -it $(shell $(DOCKER) container ls -aqf label=org.kofuk.premises.managed) bash

.PHONY: deploy-dev
deploy-dev:
	mkdir -p /tmp/premises
	CGO_ENABLED=0 GOOS=linux $(GO) build -o /tmp/premises/premises-runner
	cp debug_env.sh /tmp/premises/env

.PHONY: clean
clean:
	$(RM) premises-runner
