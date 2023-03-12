GOCMD=go
GOBUILD=$(GOCMD) build
PATH := "${CURDIR}/bin:$(PATH)"

.PHONY: gobuildcache

bin/bootstrapped/bindown:
	./script/bootstrap-bindown.sh -b bin/bootstrapped
bins += bin/bootstrapped/bindown

bin/go: bin/bootstrapped/bindown bindown.yml
	$(MAKE) bin/bootstrapped/bindown
	bin/bootstrapped/bindown install $(notdir $@)
bins += bin/go

bin/bindown: gobuildcache bin/go
	$(GOBUILD) -o $@ ./cmd/bindown
bins += bin/bindown

bin/golangci-lint: bin/bindown
	bin/bindown install $(notdir $@)
bins += bin/golangci-lint

bin/goreleaser: bin/bindown
	bin/bindown install $(notdir $@)
bins += bin/goreleaser

bin/yq: bin/bindown
	bin/bindown install $(notdir $@)
bins += bin/yq

bin/mockgen: bin/bindown
	bin/bindown install $(notdir $@)
bins += bin/mockgen

bin/semver-next: bin/bindown
	bin/bindown install $(notdir $@)
bins += bin/semver-next

bin/gofumpt: bin/bindown
	bin/bindown install $(notdir $@)
bins += bin/bindown

HANDCRAFTED_REV := 082e94edadf89c33db0afb48889c8419a2cb46a9
bin/handcrafted: Makefile
	GOBIN=${CURDIR}/bin \
	go install github.com/willabides/handcrafted@$(HANDCRAFTED_REV)

GOIMPORTS_REF := 8aaa1484dc108aa23dcf2d4a09371c0c9e280f6b
bin/goimports: Makefile
	GOBIN=${CURDIR}/bin \
	go install golang.org/x/tools/cmd/goimports@$(GOIMPORTS_REF)
bins += bin/goimports

.PHONY: clean
clean:
	rm -rf $(bins) $(cleanup_extras)
