GOBIN := $(shell go env GOBIN)
ATDIR := $(shell pwd)

# 只维护 protoc
protoc:
	go run main.go make:protoc

make-route:
	go run main.go make:route

make-bean:
	go run main.go make:bean

# 调试启动
dev:protoc make-route make-bean


