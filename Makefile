GOBIN := $(shell go env GOBIN)
ATDIR := $(shell pwd)

# mac 系统更新path可能不全
export PATH := $(GOBIN):$(PATH)

build:
	go build -ldflags="-w -s" -o  $(GOBIN)/toolset ./

build-win:
	go build -ldflags="-w -s" -o  $(GOBIN)/toolset.exe ./

# toolset make:protoc -go_out=plugins=grpc:@root/generate/proto -debug=true