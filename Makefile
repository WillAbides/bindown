PATH := "${CURDIR}/bin:$(PATH)"

.PHONY: gobuildcache

bin/bootstrapped/bindown: script/bootstrap-bindown.sh
	./script/bootstrap-bindown.sh -b bin/bootstrapped
bins += bin/bootstrapped/bindown

bin/bindown: gobuildcache
	go build -o $@ ./cmd/bindown
bins += bin/bindown

bin/golangci-lint: bin/bindown
	bin/bindown install -q  $(notdir $@)
bins += bin/golangci-lint

bin/goreleaser: bin/bindown
	bin/bindown install -q  $(notdir $@)
bins += bin/goreleaser

bin/yq: bin/bindown
	bin/bindown install -q  $(notdir $@)
bins += bin/yq

bin/semver-next: bin/bindown
	bin/bindown install -q  $(notdir $@)
bins += bin/semver-next

bin/gofumpt: bin/bindown
	bin/bindown install -q  $(notdir $@)
bins += bin/bindown

bin/shellcheck: bin/bindown
	bin/bindown install -q  $(notdir $@)
bins += bin/shellcheck

bin/shfmt: bin/bindown
	bin/bindown install -q  $(notdir $@)
bins += bin/shfmt

bin/gh: bin/bindown
	bin/bindown install -q  $(notdir $@)
bins += bin/gh

bin/jq: bin/bindown
	bin/bindown install -q  $(notdir $@)
bins += bin/jq

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
