# Copyright 2021 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# The binary to build (just the basename).
BIN := conntrack-cleaner

# This repo's root import path (under GOPATH).
PKG := conntrack-cleaner

# Where to push the docker image.
REGISTRY ?= docker.io/jallirs

# Which architecture to build - see $(ALL_ARCH) for options.
ARCH ?= amd64

# This version-strategy uses git tags to set the version string
#VERSION := $(shell git describe --tags --always --dirty)
#
# This version-strategy uses a manual value to set the version string
VERSION := 1.2.11

###
### These variables should not need tweaking.
###

SRC_DIRS := cmd # directories which hold app source (not vendored)

ALL_ARCH := amd64 arm arm64 ppc64le

# Ensure that the docker command line supports the manifest images
export DOCKER_CLI_EXPERIMENTAL=enabled

# docker interactive console
INTERACTIVE := $(shell [ -t 0 ] && echo 1 || echo 0)
TTY=
ifeq ($(INTERACTIVE), 1)
    TTY=t
endif

# Set default base image dynamically for each arch
#ifeq ($(ARCH),amd64)
#    BASEIMAGE?=k8s.gcr.io/debian-iptables-amd64:v12.0.1
#endif
#ifeq ($(ARCH),arm)
#    BASEIMAGE?=k8s.gcr.io/debian-iptables-arm:v12.0.1
#endif
#ifeq ($(ARCH),arm64)
#    BASEIMAGE?=k8s.gcr.io/debian-iptables-arm64:v12.0.1
#endif
#ifeq ($(ARCH),ppc64le)
#    BASEIMAGE?=k8s.gcr.io/debian-iptables-ppc64le:v12.0.1
#endif

ARCH := amd64
BASEIMAGE := k8s.gcr.io/debian-iptables-amd64:v12.0.1

IMAGE := $(REGISTRY)/$(BIN)-$(ARCH)
MANIFEST_IMAGE := $(REGISTRY)/$(BIN)

BUILD_IMAGE ?= golang:1.16-alpine

# If you want to build all binaries, see the 'all-build' rule.
# If you want to build all containers, see the 'all-container' rule.
# If you want to build AND push all containers, see the 'all-push' rule.
all: container

build-%:
	@$(MAKE) --no-print-directory ARCH=$* build

build: bin/$(ARCH)/$(BIN)

bin/$(ARCH)/$(BIN): build-dirs
	@echo "building: $@"
	@docker pull $(BUILD_IMAGE)
	@docker run                                                            \
            -$(TTY)i                                                           \
            -u $$(id -u):$$(id -g)                                             \
            -v $$(pwd)/.cache:/.cache                                           \
            -v $$(pwd)/.go:/go                                                 \
            -v $$(pwd):/go/src/$(PKG)                                          \
            -v $$(pwd)/bin/$(ARCH):/go/bin                                     \
            -v $$(pwd)/bin/$(ARCH):/go/bin/linux_$(ARCH)                       \
            -v $$(pwd)/.go/std/$(ARCH):/usr/local/go/pkg/linux_$(ARCH)_static  \
            -w /go/src/$(PKG)                                                  \
            $(BUILD_IMAGE)                                                     \
            /bin/sh -c "                                                       \
                ARCH=$(ARCH)                                                   \
                VERSION=$(VERSION)                                             \
                PKG=$(PKG)                                                     \
                ./build/build.sh                                               \
            "

container-%:
	@$(MAKE) --no-print-directory ARCH=$* container

DOTFILE_IMAGE = $(subst :,_,$(subst /,_,$(IMAGE))-$(VERSION))

container: .container-$(DOTFILE_IMAGE)
.container-$(DOTFILE_IMAGE): bin/$(ARCH)/$(BIN) Dockerfile.in
	@sed \
            -e 's|ARG_BIN|$(BIN)|g' \
            -e 's|ARG_ARCH|$(ARCH)|g' \
            -e 's|ARG_FROM|$(BASEIMAGE)|g' \
            Dockerfile.in > .dockerfile-$(ARCH)
	@docker build --pull -t $(IMAGE):$(VERSION) -f .dockerfile-$(ARCH) .
	@docker images -q $(IMAGE):$(VERSION) > $@
	@cp .dockerfile-$(ARCH) ../dockerfile

build-dirs:
	@mkdir -p bin/$(ARCH)
	@mkdir -p .go/src/$(PKG) .go/pkg .go/bin .go/std/$(ARCH) .cache

clean: bin-clean

bin-clean:
	rm -rf .go bin .cache
