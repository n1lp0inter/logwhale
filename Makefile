BIN_DIR=./bin/

# allow passing -ldflags, etc for release builds
BUILD_ARGS ?=

.PHONY: all
all: build test

.PHONY: clean
clean:
	go clean -cache -testcache -modcache
	go mod tidy

.PHONY: build
build: generate
	go build $(BUILD_ARGS) -o $(BIN_DIR) ./...

.PHONY: test
test:
	@echo "Running Go tests..."
	go test -race ./...

.PHONY: fmt
fmt:
	@echo "Running Go format..."
	gofmt -s -l -w .

.PHONY: lint
lint:
	@echo "Linting Go code..."
	golangci-lint run ./... --disable revive

.PHONY: vet
vet:
	@echo "Vetting Go code..."
	go vet ./...

.PHONY: generate
generate:
	@echo "Generating Go code..."
	go generate ./...

.PHONY: increment-version-patch
increment-version-patch:
	@echo "Incrementing version patch..."
	@./scripts/increment_version.sh -p internal/version/VERSION
	@cat internal/version/VERSION

.PHONY: increment-version-minor
increment-version-minor:
	@echo "Incrementing version minor..."
	@./scripts/increment_version.sh -m internal/version/VERSION
	@cat internal/version/VERSION

.PHONY: increment-version-major
increment-version-major:
	@echo "Incrementing version major..."
	@./scripts/increment_version.sh -M internal/version/VERSION
	@cat internal/version/VERSION

.PHONY: set-version-prerelease
set-version-prerelease:
	@echo "Setting prerelease version..."
	@./scripts/increment_version.sh -r internal/version/VERSION
	@cat internal/version/VERSION

.PHONY: set-version-release
set-version-release:
	@echo "Setting release version..."
	@./scripts/increment_version.sh -R internal/version/VERSION
	@cat internal/version/VERSION
