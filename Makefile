APP_NAME=cargo-agent
BIN_NAME=cargo-agent
BUILD_DIR=./bin
GO_FILES=$(shell find . -name '*.go' -not -path "./vendor/*" )

run:
	@echo "Running the server"
	@go run main.go || true

deps:
	@echo "Installing dependencies"
	@go mod tidy

fmt:
	@echo "Formatting code"
	@go fmt ./...

build:
	@echo "Building app"
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BIN_NAME) main.go
	@echo "Build complete: $(BUILD_DIR)/$(BIN_NAME)"

clean:
	@echo "Cleaning up"
	@rm -rf $(BUILD_DIR)
	@echo "CleanUp complete"

stop:
	@echo "Stopping server"
	@pkill -f "go run main.go" || echo "no process found"

migrate-up:
	@echo "Running database migration"
	@dbmate up
	@echo "Migration applied"

migrate down:
	@echo "Rolling back last migration"
	@dbmate down
	@echo "Migration rolled down"

help:
	@echo "Available commands"
	@echo " make run   -----> Run the server"
	@echo " make deps  -----> Install dependencies"
	@echo " make fmt   -----> Formats the code"
	@echo " make build -----> Build the binary"
	@echo " make clean -----> Clean up the library"
	@echo " make stop  -----> Stop running server"
	