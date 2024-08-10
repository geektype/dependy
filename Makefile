VERSION := 0.0.1-dev
GOFLAGS := -ldflags "-X main.VERSION=$(VERSION)"
BUILD_DIR := bin

bin:
	GOOS=darwin GOARCH=arm64 GOMAXPROCS=32 go build $(GOFLAGS) -o $(BUILD_DIR)/dependy ./cmd/dependy

build:
	docker build --output=$(BUILD_DIR) --target=binary .
clean:
	# TODO: Figure out a way to invalidate docker build cache too...
	rm -fr $(BUILD_DIR)/


.PHONY: bin build clean
