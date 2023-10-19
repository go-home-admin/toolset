GOBIN := $(shell go env GOBIN)
ATDIR := $(shell pwd)

# mac 系统更新path可能不全
export PATH := $(GOBIN):$(PATH)

build:
	go build -ldflags="-w -s" -o  $(GOBIN)/toolset ./

build-win:
	go build -ldflags="-w -s" -o  $(GOBIN)/toolset.exe ./

build-all:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o ./build/toolset-linux ./
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o ./build/toolset-win.exe ./
	go build -ldflags="-w -s" -o ./build/toolset-mac ./

# toolset make:protoc -go_out=plugins=grpc:@root/generate/proto -debug=true