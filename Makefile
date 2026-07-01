.PHONY: build test lint clean install cross-compile

BINARY := autocap
BUILD_DIR := build

build:
	go build -o $(BINARY) ./cmd/autocap

test:
	go test -v -race -cover ./...

lint:
	go vet ./...

clean:
	rm -f $(BINARY)
	rm -rf $(BUILD_DIR)

install: build
	cp $(BINARY) /usr/local/bin/$(BINARY)

cross-compile:
	mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 ./cmd/autocap
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 ./cmd/autocap
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY)-linux-amd64 ./cmd/autocap
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe ./cmd/autocap
