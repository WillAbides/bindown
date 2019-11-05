GOCMD=go
GOBUILD=$(GOCMD) build

.PHONY: gobuildcache

bin/bindownloader: gobuildcache
	$(GOBUILD) -o $@ ./cmd/bindownloader
bins += bin/bindownloader

bin/golangci-lint: bin/bindownloader
	bin/bindownloader $@
bins += bin/golangci-lint

bin/gobin: bin/bindownloader
	bin/bindownloader $@
bins += bin/gobin

bin/goreleaser: bin/bindownloader
	bin/bindownloader $@
bins += bin/goreleaser

GOIMPORTS_REF := 8aaa1484dc108aa23dcf2d4a09371c0c9e280f6b
bin/goimports: bin/gobin
	GOBIN=${CURDIR}/bin \
	bin/gobin golang.org/x/tools/cmd/goimports@$(GOIMPORTS_REF)
bins += bin/goimports

.PHONY: clean
clean:
	rm -rf $(bins) $(cleanup_extras)
