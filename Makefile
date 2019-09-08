GOCMD=go
GOBUILD=$(GOCMD) build

bin/bindownloader:
	$(GOBUILD) -o $@ ./cmd/bindownloader
bins += bin/bindownloader

bin/golangci-lint: bin/bindownloader
	bin/bindownloader $@
bins += bin/golangci-lint
cleanup_extras += bin/golangci-lint-*-$(UNAME)-*

.PHONY: lint
lint: bin/golangci-lint
	bin/golangci-lint run

.PHONY: fmt
fmt: bin/golangci-lint
	bin/golangci-lint run  --disable-all -E goimports --fix

.PHONY: test
test:
	$(GOCMD) test -race ./...

.PHONY: tools/bin
tools/bin:
	make -C tools all

.PHONY: clean
clean:
	rm -rf $(bins) $(cleanup_extras)
