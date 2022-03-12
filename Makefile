premises:
	npm run prod
	CGO_ENABLED=0 go build -o $@ .

.PHONY: clean
clean:
	$(RM) premises
