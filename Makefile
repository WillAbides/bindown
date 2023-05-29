PATH := "${CURDIR}/bin:$(PATH)"

.PHONY: gobuildcache

bin/bootstrapped/bindown: script/bootstrap-bindown.sh
	./script/bootstrap-bindown.sh -b bin/bootstrapped

bin/bindown: gobuildcache
	go build -o $@ ./cmd/bindown

bin/golangci-lint: bin/bindown
	bin/bindown install -q  $(notdir $@)

bin/goreleaser: bin/bindown
	bin/bindown install -q  $(notdir $@)

bin/yq: bin/bindown
	bin/bindown install -q  $(notdir $@)

bin/semver-next: bin/bindown
	bin/bindown install -q  $(notdir $@)

bin/gofumpt: bin/bindown
	bin/bindown install -q  $(notdir $@)

bin/shellcheck: bin/bindown
	bin/bindown install -q  $(notdir $@)

bin/shfmt: bin/bindown
	bin/bindown install -q  $(notdir $@)

bin/gh: bin/bindown
	bin/bindown install -q  $(notdir $@)

bin/jq: bin/bindown
	bin/bindown install -q  $(notdir $@)

HANDCRAFTED_REV := 082e94edadf89c33db0afb48889c8419a2cb46a9
bin/handcrafted: Makefile
	GOBIN=${CURDIR}/bin \
	go install github.com/willabides/handcrafted@$(HANDCRAFTED_REV)
