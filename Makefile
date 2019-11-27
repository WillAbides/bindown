GOCMD=go
GOBUILD=$(GOCMD) build

.PHONY: gobuildcache

bin/bindown: gobuildcache
	$(GOBUILD) -o $@ ./cmd/bindown
bins += bin/bindown

bin/golangci-lint: bin/bindown
	bin/bindown download $@
bins += bin/golangci-lint

bin/gobin: bin/bindown
	bin/bindown download $@
bins += bin/gobin

bin/goreleaser: bin/bindown
	bin/bindown download $@
bins += bin/goreleaser

bin/semver-next: bin/bindown
	bin/bindown download $@
bins += bin/semver-next

MOCKGEN_REF := 9fa652df1129bef0e734c9cf9bf6dbae9ef3b9fa
bin/mockgen: bin/gobin
	GOBIN=${CURDIR}/bin \
	bin/gobin github.com/golang/mock/mockgen@$(MOCKGEN_REF);
bins += bin/mockgen

GOIMPORTS_REF := 8aaa1484dc108aa23dcf2d4a09371c0c9e280f6b
bin/goimports: bin/gobin
	GOBIN=${CURDIR}/bin \
	bin/gobin golang.org/x/tools/cmd/goimports@$(GOIMPORTS_REF)
bins += bin/goimports

.PHONY: clean
clean:
	rm -rf $(bins) $(cleanup_extras)
