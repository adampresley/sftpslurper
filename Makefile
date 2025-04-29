.DEFAULT_GOAL := help
.PHONY: help

VERSION=$(shell cat ./VERSION)

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Build the application
	cd cmd/sftpslurper && CGO_ENABLED=0 go build -ldflags="-X 'main.Version=${VERSION}'" -mod=mod -o ./sftpslurper .

delete-branch: ## Delete a branch. make branch="branch-name" delete-remote-branch
	git push origin --delete ${branch}
	git branch -d ${branch}

run: ## Run the application using CompileDaemon
	cd cmd/sftpslurper && air

docker-create-builder: ## Create a builder for multi-architecture builds. Only needed once per machine
	docker buildx create --name mybuilder --driver docker-container --bootstrap

docker-tag: ## Builds a docker image and tags a release. It is then pushed up to Docker. Make sure you run docker login before this
	@echo "Creating tag ${VERSION}"
	git tag -a ${VERSION} -m "Release ${VERSION}"
	git push origin ${VERSION}
	@echo "Building ${VERSION}"
	docker buildx use mybuilder
	docker buildx build -f Dockerfile --platform linux/amd64,linux/arm64 -t github.com/adampresley/sftpslurper:${VERSION} -t github.com/adampresley/sftpslurper:latest --push .

