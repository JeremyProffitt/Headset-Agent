# SAM Makefile for Lambda function
# This is called by SAM when using BuildMethod: makefile

.PHONY: build-TroubleshootingFunction

# ARTIFACTS_DIR is set by SAM during build
build-TroubleshootingFunction:
	@echo "Packaging pre-built Lambda binary for SAM..."
	mkdir -p $(ARTIFACTS_DIR)
	cp bootstrap $(ARTIFACTS_DIR)/bootstrap
	chmod +x $(ARTIFACTS_DIR)/bootstrap
