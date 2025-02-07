GO             := go
GOLINT         := golangci-lint
MAINMODULE     := sheriff
GOOS           := $(shell go env GOOS)
GIT_TAG_NAME   ?= $(shell git describe --abbrev=1 --tags 2>/dev/null || git describe --always)
SHORT_TAG_NAME ?= $(shell git describe --abbrev=0 --tags 2>/dev/null | sed -rn 's/([0-9]+(.[0-9]+){2}).*/\1/p')
BRANCH_NAME    ?= $(shell git rev-parse --abbrev-ref HEAD)
BUILDDATE      ?= $(shell date '+%Y-%m-%d')
TARGET_DIR     ?= ./bin
TEST_TIMEOUT   ?= 900s
BUILD_VERSION  ?= 0.0.0
BINARY_EXT     := ""

ifeq ($(BUILD_VERSION), 0.0.0)
	ifdef  $(SHORT_TAG_NAME)
		BUILD_VERSION = $(SHORT_TAG_NAME)
	else 
	  BUILD_VERSION = 1.0.0
	endif
endif

ifeq ($(GOOS),windows)
	BINARY_EXT = ".exe"
endif

LDFLAGS := -s -w -X main.version=$(BUILD_VERSION) -X main.buildDate=$(BUILDDATE)
VENDOR := vendor/modules.txt

default: build

.PHONY: clean
clean:
	$(GO) clean -i ./...
	rm -rf vendor 2>/dev/null
	rm -f $(TARGET_DIR)/*$(BINARY_EXT) 2>/dev/null

.PHONY: install-tools
install-tools:
	$(GO) install golang.org/x/tools/cmd/godoc@latest
	$(GO) install github.com/go-critic/go-critic/cmd/gocritic@latest
	$(GO) install golang.org/x/tools/cmd/deadcode@latest
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
		sh -s -- -b $(go env GOPATH)/bin v2.12.2

$(VENDOR):
	$(GO) mod vendor

.PHONY: build
build: $(VENDOR)
	echo "Building ..."; \
	$(GO) build -ldflags '$(LDFLAGS)' -o "$(TARGET_DIR)/$(MAINMODULE)$(BINARY_EXT)" ".";

.PHONY: lint
lint: $(VENDOR)
	mkdir ./tmp 2>/dev/null || true
	$(GOLINT) run \
		--issues-exit-code=0 \
		--output.checkstyle.path=stdout \
		--show-stats=false \
		./... \
		| \
		tee ./tmp/lintreport.xml

.PHONY: test
test: $(VENDOR)
	$(GO) test \
		-p 1 \
		-timeout $(TEST_TIMEOUT) \
		./...

.PHONY: test-short
test-short: $(VENDOR)
	$(GO) test \
		-short \
		-p 1 \
		-timeout $(TEST_TIMEOUT) \
		./...

.PHONY: test-cover
test-cover: $(VENDOR)
	mkdir ./tmp/coverage 2>/dev/null || true
	$(GO) test \
		-short \
		-p 1 \
		-timeout $(TEST_TIMEOUT) \
		-coverprofile tmp/coverage/coverage.out \
		-covermode=count \
		-json \
		./... 1>tmp/coverage/report.json \
		|| true
	$(GO) tool cover \
		-html tmp/coverage/coverage.out \
		-o tmp/coverage/coverage.html \
		|| true

.PHONY: critic
critic: $(VENDOR)
	gocritic check \
		-enableAll \
		-disable=#experimental,whyNoLint,importShadow \
		./...

.PHONY: deadcode
deadcode: $(VENDOR)
	deadcode -test ./...

.PHONY: doc
doc: $(VENDOR)
	godoc -http=:8085 -index

.PHONY: print-%
print-%:
	@echo '$($*)'
