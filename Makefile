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

bin/jq: bin/bindownloader
	bin/bindownloader $@
bins += bin/jq

SEMREL_REF := fc68637a9654727966252b9ee358fa002e87f62f
bin/semrel: bin/gobin
	GOBIN=${CURDIR}/bin \
	bin/gobin github.com/WillAbides/semrel@$(SEMREL_REF)
bins += bin/semrel

GOIMPORTS_REF := 8aaa1484dc108aa23dcf2d4a09371c0c9e280f6b
bin/goimports: bin/gobin
	GOBIN=${CURDIR}/bin \
	bin/gobin golang.org/x/tools/cmd/goimports@$(GOIMPORTS_REF)
bins += bin/goimports

.PHONY: clean
clean:
	rm -rf $(bins) $(cleanup_extras)
