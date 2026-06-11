BINPATH ?= ./agentapi
GO_CACHE_DIR ?= .cache/go-build
GO_TMP_DIR ?= .cache/go-tmp
GO_PACKAGE_PARALLELISM ?= 1

.PHONY: lint gen build

lint:
	bash -euo pipefail -c '\
		export GOCACHE="$(PWD)/$(GO_CACHE_DIR)" \
		export GOTMPDIR="$(PWD)/$(GO_TMP_DIR)" && \
		mkdir -p "$$GOCACHE" "$$GOTMPDIR" && \
		git ls-files "*.go" ":!:agentapi-plusplus/**" | xargs gofmt -l | tee /tmp/agentapi-gofmt.out && \
		test ! -s /tmp/agentapi-gofmt.out && \
		go vet -p $(GO_PACKAGE_PARALLELISM) ./... \
	'

gen:
	bash -lc '\
		tmp_file="openapi.json.gen.tmp" && \
		rm -f "$$tmp_file" && \
		go run . server --print-openapi dummy > "$$tmp_file" && \
		rm -f openapi.json && \
		mv "$$tmp_file" openapi.json && \
		bash ./version.sh \
	'

build:
	task build:go GO_BINARY="$(BINPATH)"
