VERSION=8.0.5

OS ?= $(shell go env GOOS)
ARCH ?= $(shell go env GOARCH)
TERRAFORM_ARCH=$(OS)_$(ARCH)

LOCALDIR := $(dir $(CURDIR)/$(word $(words $(MAKEFILE_LIST)),$(MAKEFILE_LIST)))
BUILDDIR ?= build

ADDFLAGS ?=
BUILDFLAGS ?= $(ADDFLAGS) -ldflags "-w -s -X main.Sha=`git rev-parse HEAD` -X main.Version=$(VERSION)" 
CGOFLAG ?= CGO_ENABLED=1

CUSTOM_WASMER_RUNTIME_DIR ?= $(LOCALDIR)/wasmer-go

ifneq ($(CUSTOM_WASMER_RUNTIME),)
	CGO_CFLAGS ?= "-I$(CUSTOM_WASMER_RUNTIME_DIR)/wasmer/packaged/include/"
	CGO_LDFLAGS ?= "-Wl,-rpath,$(CUSTOM_WASMER_RUNTIME_DIR)/target/release/ -L$(CUSTOM_WASMER_RUNTIME_DIR)/target/release/ -lwasmer_go"
	TAGS ?= --tags custom_wasmer_runtime 
endif

# Downloads custom wasmer runtime to current directory
.PHONY: get-custom-wasmer-runtime
get-custom-wasmer-runtime:
	git clone git@github.com:wasmerio/wasmer-go.git && cd wasmer-go && cargo build --release && cd ..

# Use CUSTOM_WASMER_RUNTIME=true if you have Mac M1 and wasmer-go is built in local directory
.PHONY: build
build: clean
	GOOS=$(OS) GOARCH=$(ARCH) $(CGOFLAG) CGO_CFLAGS=$(CGO_CFLAGS) CGO_LDFLAGS=$(CGO_LDFLAGS) go build -o $(BUILDDIR)/teleport-plugin-framework $(BUILDFLAGS) $(TAGS)

.PHONY: clean
clean:
	rm -rf build/*
	go clean

.PHONY: test
test: build
	yarn asbuild:test
	./build/teleport-plugin-framework test

.PHONY: asbuild
asbuild: clean
	yarn asbuild

.PHONY: asbuild-dev
asbuild-dev: 
	yarn asbuild:dev

CUSTOM_IMPORTS_TMP_DIR ?= /tmp/protoc-gen-terraform/custom-imports

.PHONY: gen-vendor-teleport
gen-vendor-teleport:
# The wrappers.proto file needed for this generator exist only inside the go mod cache,
# so we retrieve the file path for the cached proto files with go mod tools.
	$(eval API_MOD_PATH := $(shell go mod download --json github.com/gravitational/teleport/api | jq .Dir))
	$(eval PROTOBUF_MOD_PATH := $(shell go mod download --json github.com/gogo/protobuf | jq .Dir))

# In order for types.proto to find the wrappers.proto file in the mod cache above, it
# needs be imported (-I) with its full import path discoverable. To achieve this, we
# create a temp directory and move wrappers.proto into it.
#
# Ideally, protoc-gen-terraform could be updated to reroute paths in a similar way to
# gogofast with the "M" option, which we used in the main teleport repo to overcome a
# similar issue.
	@rm -rf $(CUSTOM_IMPORTS_TMP_DIR)
	@mkdir -p $(CUSTOM_IMPORTS_TMP_DIR)/github.com/gravitational/teleport/api/types/wrappers
	@cp $(API_MOD_PATH)/types/wrappers/wrappers.proto $(CUSTOM_IMPORTS_TMP_DIR)/github.com/gravitational/teleport/api/types/wrappers

	@protoc \
		-I$(API_MOD_PATH)/types \
		-I$(PROTOBUF_MOD_PATH) \
		-I$(CUSTOM_IMPORTS_TMP_DIR) \
		--plugin=./node_modules/protobuf-as/bin/protoc-gen-as \
		--as_out=vendor \
		--as_opt=enableInterop=true:targetFileName=teleport.ts:typeAliases=Event+events.OneOf \
	    types.proto events/events.proto
