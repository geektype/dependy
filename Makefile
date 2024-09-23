VERSION := 0.0.1-dev
GOFLAGS := -ldflags "-X main.VERSION=$(VERSION)"
BUILD_DIR := bin

bin:
	GOOS=darwin GOARCH=arm64 GOMAXPROCS=32 go build $(GOFLAGS) -o $(BUILD_DIR)/darwin/dependy ./cmd/dependy
	GOOS=linux GOARCH=arm64 GOMAXPROCS=32 go build $(GOFLAGS) -o $(BUILD_DIR)/linux/dependy ./cmd/dependy

build:
	docker build -f build.Dockerfile --output=$(BUILD_DIR) --target=binary .

image:
	docker build -f Dockerfile -t sweptwings/dependy:latest .
clean:
	# TODO: Figure out a way to invalidate docker build cache too...
	rm -fr $(BUILD_DIR)/


.PHONY: bin build clean
