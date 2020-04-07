GOCMD=go
GOBUILD=$(GOCMD) build
PATH := bin:$(PATH)

.PHONY: gobuildcache

bin/bootstrapped/bindown:
	./script/bootstrap-bindown.sh -b bin/bootstrapped
bins += bin/bootstrapped/bindown

bin/go:
	$(MAKE) bin/bootstrapped/bindown
	bin/bootstrapped/bindown download $@
bins += bin/go

bin/bindown: gobuildcache bin/go
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

GCOV2LCOV_REF := 4f26027bd206195bbbd82d3944cd328a5e8dea60
bin/gcov2lcov: bin/gobin
	GOBIN=${CURDIR}/bin \
	bin/gobin github.com/jandelgado/gcov2lcov@$(GCOV2LCOV_REF)
bins += bin/gcov2lcov

GOIMPORTS_REF := 8aaa1484dc108aa23dcf2d4a09371c0c9e280f6b
bin/goimports: bin/gobin
	GOBIN=${CURDIR}/bin \
	bin/gobin golang.org/x/tools/cmd/goimports@$(GOIMPORTS_REF)
bins += bin/goimports

.PHONY: clean
clean:
	rm -rf $(bins) $(cleanup_extras)
