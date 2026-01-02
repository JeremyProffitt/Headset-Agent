.PHONY: build

build:
	@if [ -f bootstrap ]; then \
		echo "Using pre-built bootstrap binary"; \
		cp bootstrap $(ARTIFACTS_DIR)/bootstrap; \
	else \
		echo "Building bootstrap binary..."; \
		GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o $(ARTIFACTS_DIR)/bootstrap .; \
	fi
	chmod +x $(ARTIFACTS_DIR)/bootstrap
