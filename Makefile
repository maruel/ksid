# Test, lint, and format the repository.

.DEFAULT_GOAL := help
.PHONY: help test fix verify custom-gcl

# The custom-gcl binary is not byte-reproducible (golangci-lint custom builds
# in a random temp directory and stamps VCS metadata), so staleness is tracked
# by hashing the build inputs instead of comparing mtimes: the plugin config
# plus the pinned golangci-lint version and the Go toolchain.
custom-gcl:
	@version=$$(go list -m -f '{{.Version}}' github.com/golangci/golangci-lint/v2) || exit 1; \
	want=$$({ sha256sum .custom-gcl.yml | cut -d" " -f1; echo "$$version"; go env GOVERSION; } | sha256sum | cut -d" " -f1); \
	if [ -x custom-gcl ] && [ "$$want" = "$$(cat .custom-gcl.sha 2>/dev/null)" ]; then exit 0; fi; \
	echo 'Building custom-gcl with the methodfilecheck and commentcheck plugins (one-off; runs when the config, golangci-lint version, or Go toolchain changes)...'; \
	go tool golangci-lint custom --version "$$version" && echo "$$want" > .custom-gcl.sha

# The one static gate. custom-gcl run also checks the gofmt and goimports
# formatters configured in .golangci.yml.
verify: custom-gcl
	@go mod tidy -diff
	@go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
	@./custom-gcl run --show-stats=false ./...
	@go tool addlicense -ignore '**/testdata/**' -check .
	@if git ls-files -s | awk '$$1 == "100755" { print $$4; found = 1 } END { exit !found }'; then echo 'Do not commit executables'; exit 1; fi

fix: custom-gcl
	@go mod tidy
	@./custom-gcl run --show-stats=false ./... --fix
	@go tool golangci-lint fmt

test:
	@go test -timeout=600s -covermode=count -coverprofile coverage.txt -bench=. -benchtime=1x ./...

help:
	@echo 'ksid - sortable, compact time-based IDs'
	@echo ''
	@echo 'Available targets:'
	@printf '  %-14s - %s\n' 'make fix' 'Apply every autofix'
	@printf '  %-14s - %s\n' 'make verify' 'Static gate: golangci-lint, go mod tidy, actionlint, licenses, executables'
	@printf '  %-14s - %s\n' 'make test' 'Run Go tests with coverage and benchmark smoke checks'
