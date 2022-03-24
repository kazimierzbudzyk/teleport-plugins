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

# .PHONY: gen-vendor-teleport
# gen-vendor-teleport:
# 	protoc -I/Users/gzigzigzeo/go/src/github.com/gravitational/teleport/api/types -I/Users/gzigzigzeo/go/src/github.com/gravitational/teleport/api/types/wrappers -I/Users/gzigzigzeo/go/src/github.com/gravitational/teleport/vendor/github.com/gogo/protobuf/ -I/Users/gzigzigzeo/go/src --plugin=./bin/protoc-gen-as  --as_out=tmp/assembly --plugin node_modules/.bin/protoc-gen-ts_proto --as_opt enableInterop=true:targetFileName=teleport.ts --ts_proto_opt=outputJsonMethods=false,outputTypeRegistry=false,outputClientImpl=false,outputServices=false,addGrpcMetadata=false --ts_proto_out=tmp types.proto events/events.proto && yarn asc tmp/assembly/teleport.ts --tsdFile tmp/assembly/teleport.d.ts
