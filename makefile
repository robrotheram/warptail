# Define variables
APP_NAME := warptail
UI_DIR := dashboard
UI_BUILD_DIR := $(UI_DIR)/build
CRD_OUTPUT_DIR := manifests/crd
GO_SRC := ./...
VERSION := $(shell git describe --tags --always)

# Tools
CONTROLLER_GEN := controller-gen

.PHONY: all build ui crd clean run

all: build

# Build the UI and embed it into the Go binary
ui:
	@echo "Building the UI..."
	cd $(UI_DIR) && npm install && npm run build
	@echo "UI built successfully."

# Generate Kubernetes CRDs using controller-gen
crd:
	@echo "Generating Kubernetes CRDs..."
	$(CONTROLLER_GEN) crd:crdVersions=v1 output:crd:dir=$(CRD_OUTPUT_DIR) paths=$(GO_SRC)
	@echo "CRDs generated at $(CRD_OUTPUT_DIR)."

# Build the Go binary, embedding the UI
build: ui
	@echo "Building Go binary..."
	go build -ldflags "-X 'main.version=$(VERSION)'" -o $(APP_NAME) main.go
	@echo "$(APP_NAME) built successfully."

# Clean up build artifacts
clean:
	@echo "Cleaning up..."
	rm -rf $(APP_NAME) $(UI_BUILD_DIR) $(CRD_OUTPUT_DIR)
	@echo "Cleanup completed."

# Run the application
run: build
	@echo "Running $(APP_NAME)..."
	./$(APP_NAME)
