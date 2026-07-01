.PHONY: build test lint clean install cross-compile

BINARY := capauto
BUILD_DIR := build

build:
	go build -o $(BINARY) ./cmd/capauto

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
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 ./cmd/capauto
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 ./cmd/capauto
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY)-linux-amd64 ./cmd/capauto
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe ./cmd/capauto
