.DEFAULT_GOAL := help

.PHONY: help
# https://qiita.com/itoi10/items/5766df81fa28348f3fad
help: ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: fmt
fmt: ## Format
	@go fmt ./...

