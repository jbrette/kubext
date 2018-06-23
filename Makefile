PACKAGE=github.com/jbrette/kubext
CURRENT_DIR=$(shell pwd)
DIST_DIR=${CURRENT_DIR}/dist
ARGO_CLI_NAME=kubext

VERSION=$(shell cat ${CURRENT_DIR}/VERSION)
BUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT=$(shell git rev-parse HEAD)
GIT_TAG=$(shell if [ -z "`git status --porcelain`" ]; then git describe --exact-match --tags HEAD 2>/dev/null; fi)
GIT_TREE_STATE=$(shell if [ -z "`git status --porcelain`" ]; then echo "clean" ; else echo "dirty"; fi)
PACKR_CMD=$(shell if [ "`which packr`" ]; then echo "packr"; else echo "go run vendor/github.com/gobuffalo/packr/packr/main.go"; fi)

BUILDER_IMAGE=kubext-builder
# NOTE: the volume mount of ${DIST_DIR}/pkg below is optional and serves only
# to speed up subsequent builds by caching ${GOPATH}/pkg between builds.
BUILDER_CMD=docker run --rm \
  -v ${CURRENT_DIR}:/root/go/src/${PACKAGE} \
  -v ${DIST_DIR}/pkg:/root/go/pkg \
  -w /root/go/src/${PACKAGE} ${BUILDER_IMAGE}

override LDFLAGS += \
  -X ${PACKAGE}.version=${VERSION} \
  -X ${PACKAGE}.buildDate=${BUILD_DATE} \
  -X ${PACKAGE}.gitCommit=${GIT_COMMIT} \
  -X ${PACKAGE}.gitTreeState=${GIT_TREE_STATE}

# docker image publishing options
DOCKER_PUSH=false
IMAGE_TAG=latest

ifneq (${GIT_TAG},)
IMAGE_TAG=${GIT_TAG}
override LDFLAGS += -X ${PACKAGE}.gitTag=${GIT_TAG}
endif
ifneq (${IMAGE_NAMESPACE},)
override LDFLAGS += -X ${PACKAGE}/cmd/kubext/commands.imageNamespace=${IMAGE_NAMESPACE}
endif
ifneq (${IMAGE_TAG},)
override LDFLAGS += -X ${PACKAGE}/cmd/kubext/commands.imageTag=${IMAGE_TAG}
endif

ifeq (${DOCKER_PUSH},true)
ifndef IMAGE_NAMESPACE
$(error IMAGE_NAMESPACE must be set to push images (e.g. IMAGE_NAMESPACE=jbrette))
endif
endif

ifdef IMAGE_NAMESPACE
IMAGE_PREFIX=${IMAGE_NAMESPACE}/
endif

# Build the project
.PHONY: all
all: cli cli-image controller-image executor-image

.PHONY: builder
builder:
	docker build -t ${BUILDER_IMAGE} -f Dockerfile-builder .

.PHONY: cli
cli:
	CGO_ENABLED=0 ${PACKR_CMD} build -v -i -ldflags '${LDFLAGS}' -o ${DIST_DIR}/${ARGO_CLI_NAME} ./cmd/kubext

.PHONY: cli-linux
cli-linux: builder
	${BUILDER_CMD} make cli \
		CGO_ENABLED=0 \
		IMAGE_TAG=$(IMAGE_TAG) \
		IMAGE_NAMESPACE=$(IMAGE_NAMESPACE) \
		LDFLAGS='-extldflags "-static"' \
		ARGO_CLI_NAME=kubext-linux-amd64

.PHONY: cli-darwin
cli-darwin: builder
	${BUILDER_CMD} make cli \
		GOOS=darwin \
		IMAGE_TAG=$(IMAGE_TAG) \
		IMAGE_NAMESPACE=$(IMAGE_NAMESPACE) \
		ARGO_CLI_NAME=kubext-darwin-amd64

.PHONY: cli-windows
cli-windows: builder
	${BUILDER_CMD} make cli \
                GOARCH=amd64 \
		GOOS=windows \
		IMAGE_TAG=$(IMAGE_TAG) \
		IMAGE_NAMESPACE=$(IMAGE_NAMESPACE) \
		LDFLAGS='-extldflags "-static"' \
		ARGO_CLI_NAME=kubext-windows-amd64

.PHONY: controller
controller:
	go build -v -i -ldflags '${LDFLAGS}' -o ${DIST_DIR}/managed-controller ./cmd/managed-controller

.PHONY: cli-image
cli-image: cli-linux
	docker build -t $(IMAGE_PREFIX)kubextcli:$(IMAGE_TAG) -f Dockerfile-cli .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then docker push $(IMAGE_PREFIX)kubextcli:$(IMAGE_TAG) ; fi

.PHONY: controller-linux
controller-linux: builder
	${BUILDER_CMD} make controller

.PHONY: controller-image
controller-image: controller-linux
	docker build -t $(IMAGE_PREFIX)managed-controller:$(IMAGE_TAG) -f Dockerfile-managed-controller .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then docker push $(IMAGE_PREFIX)managed-controller:$(IMAGE_TAG) ; fi

.PHONY: executor
executor:
	go build -v -i -ldflags '${LDFLAGS}' -o ${DIST_DIR}/kubextexec ./cmd/kubextexec

.PHONY: executor-linux
executor-linux: builder
	${BUILDER_CMD} make executor

.PHONY: executor-image
executor-image: executor-linux
	docker build -t $(IMAGE_PREFIX)kubextexec:$(IMAGE_TAG) -f Dockerfile-kubextexec .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then docker push $(IMAGE_PREFIX)kubextexec:$(IMAGE_TAG) ; fi

.PHONY: lint
lint:
	gometalinter --config gometalinter.json ./...

.PHONY: test
test:
	go test ./...

.PHONY: update-codegen
update-codegen:
	./hack/update-codegen.sh
	./hack/update-openapigen.sh
	go run ./hack/gen-openapi-spec/main.go ${VERSION} > ${CURRENT_DIR}/api/openapi-spec/swagger.json

.PHONY: verify-codegen
verify-codegen:
	./hack/verify-codegen.sh
	./hack/update-openapigen.sh --verify-only

.PHONY: clean
clean:
	-rm -rf ${CURRENT_DIR}/dist

.PHONY: precheckin
precheckin: test lint verify-codegen

.PHONY: release-precheck
release-precheck:
	@if [ "$(GIT_TREE_STATE)" != "clean" ]; then echo 'git tree state is $(GIT_TREE_STATE)' ; exit 1; fi
	@if [ -z "$(GIT_TAG)" ]; then echo 'commit must be tagged to perform release' ; exit 1; fi

.PHONY: release
release: release-precheck controller-image cli-darwin cli-linux executor-image cli-image
