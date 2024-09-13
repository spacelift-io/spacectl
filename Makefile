include .env

lint:
	docker run --rm -v $(PWD):/app -w /app golangci/golangci-lint:$(GOLANGCI_LINT_VERSION)-alpine golangci-lint run
