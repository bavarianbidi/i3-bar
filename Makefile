.PHONY: build
build: ## Build i3-bar
	go build -o i3-bar 

FINELOG ?= mod:shell,mod:netinfo,github.com/bavarianbidi/i3-bar/shelly
LOGFILE ?= /tmp/i3-bar-debug.log

.PHONY: build-debug
build-debug: ## Build i3-bar with barista debug logging enabled
	go build -tags baristadebuglog -o i3-bar .

.PHONY: run-debug
run-debug: build-debug ## Run i3-bar with finelog; override FINELOG/LOGFILE as needed
	./i3-bar --finelog="$(FINELOG)" 2>>"$(LOGFILE)"

.PHONY: imports
imports: ## Runs goimports.
	@echo "====> $@"
	goimports -local $(MODULE) -w .

.PHONY: lint
lint: ## Runs golangci-lint.
	@echo "====> $@"
	golangci-lint run -E gosec -E goconst --timeout=15m ./...

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z%\\\/_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

# comment
# mdi-images: https://pictogrammers.com/library/mdi/