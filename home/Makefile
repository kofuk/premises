premises:
	go build -a -ldflags '-linkmode external -extldflags "-static"' -o $@ .

.PHONY: clean
clean:
	$(RM) premises
