package commands

import (
	"encoding/json"
	"fmt"
	"github.com/ctfang/command"
	"github.com/go-home-admin/toolset/console/commands/openapi"
	"github.com/go-home-admin/toolset/parser"
	"io/ioutil"
	"os"
	path2 "path"
	"strconv"
	"strings"
)

// SwaggerCommand @Bean
type SwaggerCommand struct{}

func (SwaggerCommand) Configure() command.Configure {
	return command.Configure{
		Name:        "make:swagger",
		Description: "生成文档",
		Input: command.Argument{
			Option: []command.ArgParam{
				{
					Name:        "path",
					Description: "只解析指定目录",
					Default:     "@root/protobuf",
				},
				{
					Name:        "out_route",
					Description: "生成文件到指定目录",
					Default:     "@root/web/swagger.json",
				},
			},
		},
	}
}

func (SwaggerCommand) Execute(input command.Input) {
	root := getRootPath()
	out := input.GetOption("out_route")
	out = strings.Replace(out, "@root", root, 1)

	path := input.GetOption("path")
	path = strings.Replace(path, "@root", root, 1)

	swagger := openapi.Spec{
		Swagger: "2.0",
		Info: openapi.Info{
			Title:       "go-home-admin",
			Description: "面向工程的框架",
			Version:     "1.0",
		},
		Host:     "github.com/go-home-admin",
		Schemes:  []string{"https"},
		BasePath: "/",
		Produces: []string{"application/json"},
		Paths:    make(map[string]*openapi.Path),
		Definitions: map[string]*openapi.Schema{
			"google.protobuf.Any": {
				Type: "object",
			},
		},
		Parameters:    nil,
		Extensions:    nil,
		GlobalOptions: nil,
	}

	allProtoc := parser.NewProtocParserForDir(path)
	for s, parsers := range allProtoc {
		prefix := getPrefix(path, s)
		for _, fileParser := range parsers {
			for _, message := range fileParser.Messages {
				name, parameter := messageToSchemas(prefix, message, &swagger)
				swagger.Definitions[defName(name)] = parameter
			}
			for _, enum := range fileParser.Enums {
				name, parameter := enumToMessage(prefix, enum)
				swagger.Definitions[defName(name)] = parameter
			}

			for _, service := range fileParser.Services {
				for _, rpc := range service.Rpc {
					rpcToPath(prefix, rpc, &swagger, parsers, allProtoc, service.Opt)
				}
			}
		}
	}
	by, err := json.Marshal(swagger)
	if !parser.DirIsExist(path2.Dir(out)) {
		_ = os.MkdirAll(path2.Dir(out), 0760)
	}
	err = ioutil.WriteFile(out, by, 0766)
	if err != nil {
		fmt.Println("gen openapi.json err " + err.Error() + ", out = " + out)
	} else {
		fmt.Println("gen openapi.json to " + out)
	}
}

func defName(name string) string {
	arr := strings.Split(name, ".")
	if len(arr) > 2 {
		str := ""
		for i := len(arr) - 1; i < len(arr); i++ {
			str += arr[i-1] + "."
		}
		str += arr[len(arr)-1]
		return str
	}
	return name
}

func rpcToPath(pge string, service parser.ServiceRpc, swagger *openapi.Spec, nowDirProtoc []parser.ProtocFileParser, allProtoc map[string][]parser.ProtocFileParser, serviceOpt map[string]parser.Option) {
	for _, option := range service.Opt {
		urlPath := option.Val
		if routeGroup, ok := serviceOpt["http.RouteGroup"]; ok {
			urlPath = "$[" + routeGroup.Val + "]" + urlPath
		}
		var path = &openapi.Path{}
		if o, ok := swagger.Paths[urlPath]; ok {
			path = o
		}

		endpoint := &openapi.Endpoint{}
		endpoint.Description = service.Doc + option.Doc
		endpoint.Summary = service.Doc + option.Doc
		endpoint.Tags = strings.Split(pge, ".")
		endpoint.Parameters = messageToParameters(service.Param, nowDirProtoc, allProtoc)
		endpoint.Responses = map[string]*openapi.Response{
			"200": messageToResponse(service.Return, nowDirProtoc, allProtoc),
		}

		switch option.Key {
		case "http.Get":
			path.Get = endpoint
		case "http.Put":
			path.Put = endpoint
		case "http.Post":
			path.Post = endpoint
		case "http.Patch":
			path.Patch = endpoint
		case "http.Delete":
			path.Delete = endpoint
		}

		swagger.Paths[urlPath] = path
	}
}

func messageToResponse(message string, nowDirProtoc []parser.ProtocFileParser, allProtoc map[string][]parser.ProtocFileParser) *openapi.Response {
	protocMessage, pge := findMessage(message, nowDirProtoc, allProtoc)
	got := &openapi.Response{
		Description: protocMessage.Doc,
		Schema: &openapi.Schema{
			Ref: "#/definitions/" + pge + "." + protocMessage.Name,
		},
	}

	return got
}

func messageToParameters(message string, nowDirProtoc []parser.ProtocFileParser, allProtoc map[string][]parser.ProtocFileParser) openapi.Parameters {
	protocMessage, pge := findMessage(message, nowDirProtoc, allProtoc)
	got := openapi.Parameters{}

	for _, option := range protocMessage.Attr {
		if option.Repeated {
			if isProtoBaseType(option.Ty) {
				// 基础类型的数组
				attr := &openapi.Parameter{
					Name:        option.Name,
					Description: option.Doc,
					Enum:        nil,
					Format:      option.Ty,
					In:          "query",
					Items: &openapi.Schema{
						Description: getTitle(option.Doc),
						Type:        getProtoToSwagger(option.Ty),
						Format:      option.Ty,
					},
					Type: "array",
				}
				got = append(got, attr)
			} else {
				// 引用其他对象
				attr := &openapi.Parameter{
					Name: option.Name,
					Type: "array",
					In:   "query",
					Items: &openapi.Schema{
						Ref:         getRef(pge, option.Ty),
						Description: getTitle(option.Doc),
						Type:        "object",
						Format:      option.Ty,
					},
				}
				got = append(got, attr)
			}
		} else if isProtoBaseType(option.Ty) {
			attr := &openapi.Parameter{
				Name:        option.Name,
				In:          "query",
				Description: getTitle(option.Doc),
				Type:        getProtoToSwagger(option.Ty),
				Format:      option.Ty,
			}
			got = append(got, attr)
		} else {
			// 引用其他对象
			attr := &openapi.Parameter{
				Name:        option.Name,
				Description: getTitle(option.Doc),
				Type:        getProtoToSwagger(option.Ty),
				Format:      option.Ty,
				In:          "query",
				Schema: &openapi.Schema{
					Type:        "object",
					Description: getTitle(option.Doc),
					Ref:         getRef(pge, option.Ty),
				},
			}

			got = append(got, attr)
		}
	}

	return got
}

func getRef(pge string, ty string) string {
	arr := strings.Split(ty, ".")
	if len(arr) == 1 {
		return "#/definitions/" + pge + "." + ty
	}

	return "#/definitions/" + ty
}

func messageToSchemas(pge string, message parser.Message, swagger *openapi.Spec) (string, *openapi.Schema) {
	schema := &openapi.Schema{}
	schema.Description = message.Doc
	properties := make(map[string]*openapi.Schema)

	for _, option := range message.Attr {
		if option.Repeated {
			if isProtoBaseType(option.Ty) {
				// 基础类型的数组
				attr := &openapi.Schema{
					Type: "array",
					Items: &openapi.Schema{
						Description: getTitle(option.Doc),
						Type:        getProtoToSwagger(option.Ty),
						Format:      option.Ty,
					},
				}
				properties[option.Name] = attr
			} else if option.Message != nil {
				name, parameter := messageToSchemas(pge, *option.Message, swagger)
				name = pge + "." + option.Name + "_" + name
				swagger.Definitions[defName(name)] = parameter
				attr := &openapi.Schema{
					Description: getTitle(option.Doc),
					Ref:         "#/definitions/" + defName(name), // 嵌套肯定是本包
				}
				properties[option.Name] = attr
			} else {
				// 引用其他对象
				attr := &openapi.Schema{
					Type: "array",
					Items: &openapi.Schema{
						Ref:         getRef(pge, option.Ty),
						Description: getTitle(option.Doc),
						Type:        "object",
						Format:      option.Ty,
					},
				}
				properties[option.Name] = attr
			}
		} else if isProtoBaseType(option.Ty) {
			attr := &openapi.Schema{
				Description: getTitle(option.Doc),
				Type:        getProtoToSwagger(option.Ty),
				Format:      option.Ty,
			}
			properties[option.Name] = attr
		} else if option.Message != nil {
			name, parameter := messageToSchemas(pge, *option.Message, swagger)
			name = pge + "." + option.Name + "_" + name
			swagger.Definitions[defName(name)] = parameter
			attr := &openapi.Schema{
				Description: getTitle(option.Doc),
				Ref:         "#/definitions/" + defName(name), // 嵌套肯定是本包
			}
			properties[option.Name] = attr
		} else {
			attr := &openapi.Schema{
				Description: getTitle(option.Doc),
				Ref:         getRef(pge, option.Ty),
			}
			properties[option.Name] = attr
		}
	}

	schema.Type = "object"
	schema.Properties = properties
	return pge + "." + message.Name, schema
}

func enumToMessage(pge string, enum parser.Enum) (string, *openapi.Schema) {
	schema := &openapi.Schema{}
	schema.Description = enum.Doc
	properties := make(map[string]*openapi.Schema)
	for i, opt := range enum.Opt {
		attr := &openapi.Schema{
			Description: "enum|" + getTitle(opt.Doc),
			Type:        "number",
			Format:      "uint",
		}
		properties[strconv.Itoa(i)] = attr
	}
	schema.Properties = properties
	schema.Format = "number"
	schema.Type = "object"
	return pge + "." + enum.Name, schema
}

func getTitle(str string) string {
	str = strings.ReplaceAll(str, ";", "")
	str = strings.ReplaceAll(str, "//", "")
	str = strings.ReplaceAll(str, "=", "")

	return str
}

var protoToSwagger = map[string]string{
	"double":   "number",
	"float":    "number",
	"int32":    "integer",
	"int64":    "integer",
	"uint32":   "number",
	"uint64":   "number",
	"fixed32":  "number",
	"fixed64":  "number",
	"sfixed32": "number",
	"sfixed64": "number",
	"bool":     "boolean",
	"string":   "string",
	"bytes":    "string",
}

func getProtoToSwagger(t string) string {
	ty, ok := protoToSwagger[t]
	if ok {
		return ty
	}
	return "string"
}

func getPrefix(path, s string) string {
	got := strings.ReplaceAll(s, path, "")
	got = strings.Trim(got, "/")
	got = strings.ReplaceAll(got, "/", ".")

	return got
}

func isProtoBaseType(str string) bool {
	switch str {
	case "double", "float", "int32", "int64", "uint32", "uint64", "fixed32", "fixed64", "sfixed32", "sfixed64", "bool", "string", "bytes":
		return true
	case "google.protobuf.Any":
		return false
	default:
		return false
	}
}

// 查找message, pge 当前package, name 名称
func findMessage(message string, nowDirProtoc []parser.ProtocFileParser, allProtoc map[string][]parser.ProtocFileParser) (*parser.Message, string) {
	for _, fileParser := range nowDirProtoc {
		for _, m := range fileParser.Messages {
			if message == m.Name {
				return &m, fileParser.PackageName
			}
		}
	}
	// 其他包也尝试查询
	arr := strings.Split(message, ".")
	if len(arr) >= 2 {
		for _, parsers := range allProtoc {
			for _, p := range parsers {
				if arr[0] == p.PackageName {
					for _, m := range p.Messages {
						if message == m.Name {
							return &m, p.PackageName
						}
					}
				}
			}
		}
	} else {
		for _, parsers := range allProtoc {
			for _, p := range parsers {
				for _, m := range p.Messages {
					if message == m.Name {
						return &m, p.PackageName
					}
				}
			}
		}
	}
	return nil, ""
}
