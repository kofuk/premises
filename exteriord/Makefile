exteriord:
	CGO_ENABLED=0 GOOS=linux go build -o $@

.PHONY: deploy-dev
deploy-dev:
	mkdir -p /tmp/premises
	CGO_ENABLED=0 GOOS=linux go build -o /tmp/premises/exteriord
