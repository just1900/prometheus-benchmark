# All these commands must run from repository root.

DOCKER_NAMESPACE := ghcr.io/just1900
GO_BUILDER_IMAGE := golang:1.18.1-alpine

BUILDINFO_TAG ?= $(shell echo $$(git describe --long --all | tr '/' '-')$$( \
	      git diff-index --quiet HEAD -- || echo '-dirty-'$$(git diff-index -u HEAD | openssl sha1 | cut -c 10-17)))

PKG_TAG ?= $(shell git tag -l --points-at HEAD)
ifeq ($(PKG_TAG),)
PKG_TAG := $(BUILDINFO_TAG)
endif

stress-querier-prod:
	mkdir -p gocache-for-docker
	docker run --rm \
		--user $(shell id -u):$(shell id -g) \
		--mount type=bind,src="$(shell pwd)",dst=/prometheus-benchmark \
		-w /prometheus-benchmark \
		--mount type=bind,src="$(shell pwd)/gocache-for-docker",dst=/gocache \
		--env GOCACHE=/gocache \
		--env GO111MODULE=on \
		--env CGO_ENABLED=0 \
		--env GOOS=linux \
		--env GOARCH=amd64 \
		$(GO_BUILDER_IMAGE) \
		go build -mod=vendor -trimpath -buildvcs=false \
			-ldflags "-s -w -extldflags '-static'" \
			-tags 'netgo osusergo nethttpomithttp2' \
			-o bin/stress-querier-prod ./services/stress-querier/

stress-querier-package: stress-querier-prod
	(docker image ls --format '{{.Repository}}:{{.Tag}}' | grep -q '$(DOCKER_NAMESPACE)/stress-querier:$(PKG_TAG)$$') || (\
		$(MAKE) stress-querier-prod && \
		docker build \
			--build-arg src_binary=stress-querier-prod \
			--tag $(DOCKER_NAMESPACE)/stress-querier:$(PKG_TAG) \
			-f services/stress-querier/Dockerfile bin)

stress-querier-publish: stress-querier-package
	docker push $(DOCKER_NAMESPACE)/stress-querier:$(PKG_TAG)

stress-querier-package-run: stress-querier-package
	docker run -it --rm \
		--user $(shell id -u):$(shell id -g) \
		--net host \
		$(DOCKER_NAMESPACE)/stress-querier:$(PKG_TAG)

stress-querier-run: stress-querier-prod
	bin/stress-querier-prod
