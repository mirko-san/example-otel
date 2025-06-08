.DEFAULT_GOAL := help

GO_VERSION := 1.24.3

# BUILD_COMMAND を docker にしたら docker で動くかも
BUILD_COMMAND := buildah
REGISTORY_ENDPOINT := docker://localhost:5000

.PHONY: help
# https://qiita.com/itoi10/items/5766df81fa28348f3fad
help: ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: fmt
fmt: ## Format
	@go fmt ./...

# image
.PHONY: image-build
image-build: image-build-server image-build-client ## Build All Image

image-build-server image-build-client: image-build-%:
	@${BUILD_COMMAND} build \
		--format=docker \
		-f dockerfiles/$*.Dockerfile \
		-t example-otel/$*:latest \
		--build-arg="GO_VERSION=${GO_VERSION}" \
		--platform=linux/amd64 \
		.

.PHONY: image-push
image-push: image-push-server image-push-client ## Push All Image

image-push-server image-push-client: image-push-%:
	@${BUILD_COMMAND} push \
		--tls-verify=false \
		localhost/example-otel/$* \
		${REGISTORY_ENDPOINT}/example-otel/$*:latest
