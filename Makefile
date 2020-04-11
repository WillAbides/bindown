GOCMD=go
GOBUILD=$(GOCMD) build
PATH := "${CURDIR}/bin:$(PATH)"

.PHONY: gobuildcache

bin/bootstrapped/bindown:
	./script/bootstrap-bindown.sh -b bin/bootstrapped
bins += bin/bootstrapped/bindown

bin/go:
	$(MAKE) bin/bootstrapped/bindown
	BINDOWN_CONFIG_FILE="bindown-v2.yml" \
	bin/bootstrapped/bindown download $@
bins += bin/go

bin/bindown: gobuildcache bin/go
	$(GOBUILD) -o $@ ./cmd/bindown
bins += bin/bindown

bin/golangci-lint: bin/bindown
	bin/bindown install $@
bins += bin/golangci-lint

bin/gobin: bin/bindown
	bin/bindown install $@
bins += bin/gobin

bin/goreleaser: bin/bindown
	bin/bindown install $@
bins += bin/goreleaser

bin/semver-next: bin/bindown
	bin/bindown install $@
bins += bin/semver-next

GOIMPORTS_REF := 8aaa1484dc108aa23dcf2d4a09371c0c9e280f6b
bin/goimports: bin/gobin
	GOBIN=${CURDIR}/bin \
	bin/gobin golang.org/x/tools/cmd/goimports@$(GOIMPORTS_REF)
bins += bin/goimports

.PHONY: clean
clean:
	rm -rf $(bins) $(cleanup_extras)
