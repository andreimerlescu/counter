PROJECT_NAME := counter
OUTPUT_DIR := bin
OUTPUTS_DIR := outputs
COVER_OUT := $(OUTPUTS_DIR)/coverage.out
COVER_JSON := $(OUTPUTS_DIR)/coverage.json
GO_BUILD := go build -o
TARGETS := \
	darwin/amd64 \
	darwin/arm64 \
	linux/amd64 \
	linux/arm64

# Default target
.PHONY: all
all: prepare $(TARGETS)

.PHONY: prepare
prepare:
	@go mod tidy
	@go mod download
	@find . -type f -name '*.go' -exec gofmt -w {} \;

.PHONY: coverage
coverage:
	@mkdir -p $(OUTPUTS_DIR)
	@go test -coverprofile=$(COVER_OUT)
	@go tool cover -func=$(COVER_OUT)
	@go tool cover -o $(COVER_JSON) -func=$(COVER_OUT)

.PHONY: install
install:
	@sudo rm -rf /usr/local/bin/$(PROJECT_NAME)
	@sudo cp bin/$(PROJECT_NAME)-linux-amd64 /usr/local/bin/$(PROJECT_NAME)
	@sudo chmod +x /usr/local/bin/$(PROJECT_NAME)
	@echo "Installed inside /usr/local/bin/$(PROJECT_NAME)"
	@bash -c '"/usr/local/bin/$(PROJECT_NAME)" --help'

.PHONY: uninstall
uninstall:
	@sudo rm -rf /usr/local/bin/$(PROJECT_NAME)

.PHONY: remove
remove: uninstall

.PHONY: delete
delete: uninstall

# Build targets for each OS/Arch combination
.PHONY: $(TARGETS)
$(TARGETS):
	@echo "Building for GOOS=$(word 1,$(subst /, ,$@)) GOARCH=$(word 2,$(subst /, ,$@))..."
	GOOS=$(word 1,$(subst /, ,$@)) GOARCH=$(word 2,$(subst /, ,$@)) $(GO_BUILD) $(OUTPUT_DIR)/$(PROJECT_NAME)-$(word 1,$(subst /, ,$@))-$(word 2,$(subst /, ,$@)) .

# Clean up binaries
.PHONY: clean
clean: uninstall
	@rm -rf $(OUTPUT_DIR)
	@rm -rf $(OUTPUTS_DIR)

# Run the package
.PHONY: run
run: prepare
	go run . $(ARGS)

.PHONY: test
test: prepare
	@mkdir -p $(OUTPUTS_DIR)
	@go test -json  ./... $(ARGS) > $(OUTPUTS_DIR)/tests.json 2> /dev/null
	@go test ./... $(ARGS)

# Help target
.PHONY: help
help:
	@echo "Makefile for $(PROJECT_NAME)"
	@echo
	@echo "Usage:"
	@echo "  make [target]"
	@echo
	@echo "Targets:"
	@echo "  all       Build binaries for all target OS/Arch combinations"
	@echo "  run       Run the Go code directly"
	@echo "  clean     Remove all binaries"
	@echo "  coverage  Test coverage report"
	@echo "  test      Test project"
	@echo "  help      Display this help message"
	@echo
	@echo "Target OS/Arch combinations:"
	@echo "  darwin/amd64"
	@echo "  darwin/arm64"
	@echo "  linux/amd64"
	@echo "  linux/arm64"

