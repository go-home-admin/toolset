package commands

import (
	"fmt"
	"github.com/ctfang/command"
	"github.com/go-home-admin/toolset/parser"
	"strings"
)

// GrpcCommand @Bean
type GrpcCommand struct{}

func (GrpcCommand) Configure() command.Configure {
	return command.Configure{
		Name:        "make:grpc",
		Description: "根据protoc文件定义, 生成路grpc基础文件",
		Input: command.Argument{
			Option: []command.ArgParam{
				{
					Name:        "path",
					Description: "只解析指定目录",
					Default:     "@root/protobuf/endpoint",
				},
			},
		},
	}
}

// TODO
func (GrpcCommand) Execute(input command.Input) {
	root := getRootPath()
	module := getModModule()

	_ = module

	path := input.GetOption("path")
	path = strings.Replace(path, "@root", root, 1)

	for _, parsers := range parser.NewProtocParserForDir(path) {
		for _, fileParser := range parsers {
			for _, service := range fileParser.Services {
				fmt.Println(service)
			}
		}
	}
}
