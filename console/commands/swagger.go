package commands

import (
	"encoding/json"
	"fmt"
	"github.com/ctfang/command"
	"github.com/go-home-admin/toolset/console/commands/openapi"
	"github.com/go-home-admin/toolset/parser"
	"os"
	path2 "path"
	"regexp"
	"strconv"
	"strings"
)

// SwaggerCommand @Bean
type SwaggerCommand struct{}

var language string

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
					Name:        "source",
					Description: "基础文件, 在这个文件上补充信息",
					Default:     "@root/web/swagger.json",
				},
				{
					Name:        "out",
					Description: "生成文件到指定目录",
					Default:     "@root/web/swagger_gen.json",
				},
				{
					Name:        "lang",
					Description: "指定文档语言，如说明的下一行是‘// @lang:en name’则‘name’代替原说明",
					Default:     "",
				},
				{
					Name:        "host",
					Description: "指定接口Host",
					Default:     "",
				},
			},
		},
	}
}

func (SwaggerCommand) Execute(input command.Input) {
	input = repRootPath(input)
	source := input.GetOption("source")
	out := input.GetOption("out")
	path := input.GetOption("path")
	language = input.GetOption("lang")
	host := input.GetOption("host")

	swagger := openapi.Spec{
		Swagger:  "2.0",
		Schemes:  []string{"http", "https"},
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
	if host != "" {
		re := regexp.MustCompile(`^(https?)://(.+)`)
		matches := re.FindStringSubmatch(host)
		if matches != nil {
			swagger.Schemes = []string{matches[1]}
			swagger.Host = matches[2]
		} else {
			swagger.Host = host
		}
	}
	if parser.DirIsExist(source) {
		data, _ := os.ReadFile(source)
		json.Unmarshal(data, &swagger)
	}

	allProtoc := parser.NewProtocParserForDir(path)
	for s, parsers := range allProtoc {
		pkg := getPackage(path, s)
		for _, fileParser := range parsers {
			for _, message := range fileParser.Messages {
				name, parameter := messageToSchemas(pkg, message, &swagger)
				swagger.Definitions[defName(name)] = parameter
			}
			for _, enum := range fileParser.Enums {
				name, parameter := enumToMessage(pkg, enum)
				swagger.Definitions[defName(name)] = parameter
			}

		}
	}
	// 全局定义后在生成url
	for s, parsers := range allProtoc {
		pkg := getPackage(path, s)
		for _, fileParser := range parsers {
			for _, service := range fileParser.Services {
				var prefix string
				if routeGroup, ok := service.Opt["http.RouteGroup"]; ok {
					prefix = "$[" + routeGroup.Val + "]"
					if routeGroup.Doc != "" {
						re := regexp.MustCompile(`(?i)@prefix=(.*)`)
						match := re.FindStringSubmatch(routeGroup.Doc)
						if match != nil {
							prefix = match[1]
							if !strings.HasPrefix(prefix, "/") {
								prefix = "/" + prefix
							}
						}
					}
				}
				for _, rpc := range service.Rpc {
					rpcToPath(pkg, rpc, &swagger, parsers, allProtoc, prefix)
				}
			}
		}
	}
	// 检查对象引用, 如果发现引用没有在定义的包，有可能是标准库等，补充个空对象
	like := make(map[string]bool)
	for _, schema := range swagger.Paths {
		for _, parameter := range schema.Parameters {
			like[parameter.Ref] = true
		}
		if schema.Get != nil {
			for _, parameter := range schema.Get.Responses {
				like[parameter.Schema.Ref] = true
			}
		}
		if schema.Post != nil {
			for _, parameter := range schema.Post.Responses {
				like[parameter.Schema.Ref] = true
			}
		}
	}
	for _, schema := range swagger.Definitions {
		for _, parameter := range schema.Properties {
			if parameter.Ref != "" {
				like[parameter.Ref] = true
			}
			if parameter.Items != nil && parameter.Items.Ref != "" {
				like[parameter.Items.Ref] = true
			}
		}
	}

	for schema, _ := range like {
		name := strings.ReplaceAll(schema, "#/definitions/", "")
		if _, ok := swagger.Definitions[name]; !ok {
			swagger.Definitions[name] = &openapi.Schema{
				Type: "object",
			}
		}
	}

	by, err := json.Marshal(swagger)
	if !parser.DirIsExist(path2.Dir(out)) {
		_ = os.MkdirAll(path2.Dir(out), 0760)
	}
	err = os.WriteFile(out, by, 0766)
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

func getUrl(opts map[string][]parser.Option) string {
	urlPath := ""
	for _, options := range opts {
		for _, option := range options {
			switch option.Key {
			case "http.Get", "http.Put", "http.Post", "http.Patch", "http.Delete":
				urlPath = option.Val
			}
		}
	}
	return urlPath
}

func isValidHTTPStatusCode(code int) bool {
	return code >= 100 && code <= 599
}

func rpcToPath(pge string, service parser.ServiceRpc, swagger *openapi.Spec, nowDirProtoc []parser.ProtocFileParser, allProtoc map[string][]parser.ProtocFileParser, prefix string) {
	service.Doc = filterLanguage(service.Doc)
	for _, options := range service.Opt {
		for _, option := range options {
			urlPath := option.Val
			if urlPath == "" {
				urlPath = getUrl(service.Opt)
			}
			if prefix != "" {
				urlPath = prefix + urlPath
			}
			var path = &openapi.Path{}
			if o, ok := swagger.Paths[urlPath]; ok {
				path = o
			}
			endpoint := &openapi.Endpoint{}
			switch option.Key {
			case "http.Get", "http.Put", "http.Post", "http.Patch", "http.Delete":
				endpoint.Description = parseTag(service.Doc)
				endpoint.Summary = filterTag(service.Doc)
				endpoint.Tags = strings.Split(pge, ".")
				endpoint.Parameters = parseParamInPath(option)
				if option.Key == "http.Get" {
					endpoint.Parameters = append(endpoint.Parameters, messageToParameters(service.Param, nowDirProtoc, allProtoc)...)
				} else {
					endpoint.RequestBody = messageToRequestBody(service.Param, nowDirProtoc, allProtoc)
				}
				endpoint.Responses = map[string]*openapi.Response{
					"200": messageToResponse(service.Return, nowDirProtoc, allProtoc),
				}
			case "http.Status":
				// 其他状态码的返回结构
				for _, endpoint := range []*openapi.Endpoint{
					path.Get,
					path.Post,
					path.Put,
					path.Patch,
					path.Delete,
				} {
					if endpoint != nil {
						code := option.Map["Code"]
						resp := option.Map["Response"]

						if _, ok := swagger.Definitions[resp]; !ok {
							// 如果不存在，搜索本包
							if _, ok := swagger.Definitions[pge+"."+resp]; ok {
								resp = pge + "." + resp
							}
						}

						codeInt, _ := strconv.Atoi(code)
						if isValidHTTPStatusCode(codeInt) {
							endpoint.Responses[code] = &openapi.Response{
								Description: option.Doc,
								Schema: &openapi.Schema{
									Ref: "#/definitions/" + resp,
								},
							}
						} else {
							// 非法的状态码，自动补充一个
							for i := 201; i < 599; i++ {
								intCode := strconv.Itoa(i)
								if _, ok := endpoint.Responses[intCode]; !ok {
									endpoint.Responses[intCode] = &openapi.Response{
										Description: fmt.Sprintf("logic(%s)", code) + option.Doc,
										Schema: &openapi.Schema{
											Ref: "#/definitions/" + resp,
										},
									}
									break
								}
							}
						}
					}
				}
			default:
				continue
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
			case "http.Any":
				path.Get = endpoint
				path.Post = endpoint
				path.Put = endpoint
				path.Patch = endpoint
				path.Delete = endpoint
			}

			swagger.Paths[urlPath] = path
		}
	}
}

func messageToResponse(message string, nowDirProtoc []parser.ProtocFileParser, allProtoc map[string][]parser.ProtocFileParser) *openapi.Response {
	protocMessage, pge := findMessage(message, nowDirProtoc, allProtoc)
	got := &openapi.Response{
		Description: protocMessage.Doc,
		Schema: &openapi.Schema{
			Type:   "object",
			Format: pge + "." + protocMessage.Name,
			Ref:    "#/definitions/" + pge + "." + protocMessage.Name,
		},
	}

	return got
}

func messageToParameters(message string, nowDirProtoc []parser.ProtocFileParser, allProtoc map[string][]parser.ProtocFileParser) openapi.Parameters {
	protocMessage, pge := findMessage(message, nowDirProtoc, allProtoc)
	got := openapi.Parameters{}
	if protocMessage == nil {
		return got
	}
	in := "query"
	for _, option := range protocMessage.Attr {
		doc, isRequired := filterRequired(option.Doc)
		doc, example := filterExample(doc, option.Ty)
		doc = filterLanguage(doc)
		doc = getTitle(doc)
		if option.Repeated {
			if isProtoBaseType(option.Ty) {
				// 基础类型的数组
				attr := &openapi.Parameter{
					Name:        option.Name,
					Description: doc,
					Enum:        nil,
					Format:      option.Ty,
					In:          in,
					Required:    isRequired,
					Example:     example,
					Items: &openapi.Schema{
						Description: doc,
						Type:        getProtoToSwagger(option.Ty),
						Format:      option.Ty,
					},
					Type: "array",
				}
				got = append(got, attr)
			} else {
				// 引用其他对象
				attr := &openapi.Parameter{
					Name:        option.Name,
					Description: doc,
					Type:        "array",
					In:          in,
					Required:    isRequired,
					Example:     example,
					Items: &openapi.Schema{
						Ref:         getRef(pge, option.Ty),
						Description: doc,
						Type:        "object",
						Format:      option.Ty,
					},
				}
				got = append(got, attr)
			}
		} else if isProtoBaseType(option.Ty) {
			attr := &openapi.Parameter{
				Name:        option.Name,
				In:          in,
				Description: doc,
				Type:        getProtoToSwagger(option.Ty),
				Format:      option.Ty,
				Required:    isRequired,
				Example:     example,
			}
			got = append(got, attr)
		} else {
			// 引用其他对象
			attr := &openapi.Parameter{
				Name:        option.Name,
				Description: doc,
				Type:        getProtoToSwagger(option.Ty),
				Format:      option.Ty,
				In:          in, // 对象引用只能是query, 不然页面显示错误
				Required:    isRequired,
				Example:     example,
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

func messageToRequestBody(message string, nowDirProtoc []parser.ProtocFileParser, allProtoc map[string][]parser.ProtocFileParser) *openapi.RequestBody {
	protocMessage, pge := findMessage(message, nowDirProtoc, allProtoc)
	if len(protocMessage.Attr) == 0 {
		return nil
	}
	doc, _ := filterRequired(protocMessage.Doc)
	doc = filterLanguage(doc)
	got := &openapi.RequestBody{
		Description: parseTag(doc),
		Content: openapi.RequestBodyContent{
			// 目前只支持application/json
			Json: &openapi.RequestBodyContentType{
				Schema: openapi.Schema{
					Ref: getRef(pge, protocMessage.Name),
				},
			},
		},
	}
	return got
}

func getRef(pge string, ty string) string {
	arr := strings.Split(ty, ".")
	if len(arr) == 1 {
		arr = strings.Split(pge, ".")
		return "#/definitions/" + arr[len(arr)-1] + "." + ty
	}

	return "#/definitions/" + ty
}

func messageToSchemas(pge string, message parser.Message, swagger *openapi.Spec) (string, *openapi.Schema) {
	schema := &openapi.Schema{}
	schema.Description = message.Doc
	properties := make(map[string]*openapi.Schema)
	var requireArr []string
	for _, option := range message.Attr {
		doc, isRequired := filterRequired(option.Doc)
		doc, example := filterExample(doc, option.Ty)
		doc = filterLanguage(doc)
		doc = getTitle(doc)
		if isRequired {
			requireArr = append(requireArr, option.Name)
		}
		if option.Repeated {
			if isProtoBaseType(option.Ty) {
				// 基础类型的数组
				attr := &openapi.Schema{
					Type:        "array",
					Description: doc,
					Items: &openapi.Schema{
						Description: doc,
						Type:        getProtoToSwagger(option.Ty),
						Format:      option.Ty,
					},
					Example: example,
				}
				properties[option.Name] = attr
			} else if option.Message != nil {
				name, parameter := messageToSchemas(pge, *option.Message, swagger)
				name = pge + "." + option.Name + "_" + name
				swagger.Definitions[defName(name)] = parameter
				attr := &openapi.Schema{
					Description: doc,
					Ref:         "#/definitions/" + defName(name), // 嵌套肯定是本包
				}
				properties[option.Name] = attr
			} else {
				// 引用其他对象
				attr := &openapi.Schema{
					Type:        "array",
					Description: doc,
					Example:     example,
					Items: &openapi.Schema{
						Ref:         getRef(pge, option.Ty),
						Description: doc,
						Type:        "object",
						Format:      option.Ty,
					},
				}
				properties[option.Name] = attr
			}
		} else if isProtoBaseType(option.Ty) {
			attr := &openapi.Schema{
				Description: doc,
				Type:        getProtoToSwagger(option.Ty),
				Format:      option.Ty,
				Example:     example,
			}
			properties[option.Name] = attr
		} else if option.Message != nil {
			name, parameter := messageToSchemas(pge, *option.Message, swagger)
			name = pge + "." + option.Name + "_" + name
			swagger.Definitions[defName(name)] = parameter
			attr := &openapi.Schema{
				Description: doc,
				Ref:         "#/definitions/" + defName(name), // 嵌套肯定是本包
			}
			properties[option.Name] = attr
		} else {
			if doc == "" {
				//使用ref的Description
				properties[option.Name] = &openapi.Schema{
					Description: doc,
					Ref:         getRef(pge, option.Ty),
				}
			} else {
				//使用本地的Description
				properties[option.Name] = &openapi.Schema{
					AllOf: []*openapi.Schema{
						{
							Description: doc,
						},
						{
							Format: option.Ty,
						},
						{
							Ref: getRef(pge, option.Ty),
						},
					},
				}
			}
		}
	}

	schema.Type = "object"
	schema.Properties = properties
	schema.Required = requireArr
	return pge + "." + message.Name, schema
}

func enumToMessage(pge string, enum parser.Enum) (string, *openapi.Schema) {
	schema := &openapi.Schema{}
	schema.Description = enum.Doc
	properties := make(map[string]*openapi.Schema)
	for _, opt := range enum.Opt {
		attr := &openapi.Schema{
			Description: "enum|" + getTitle(filterLanguage(opt.Doc)),
			Type:        opt.Name,
			Format:      "number",
		}
		properties[fmt.Sprint(opt.Num)] = attr
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
	"int":      "integer",
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
	return "object"
}

func getPackage(path, s string) string {
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
						if arr[1] == m.Name {
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

func filterRequired(doc string) (string, bool) {
	arr := strings.Split(doc, "\n")
	var newArr []string
	var isRequired bool
	re := regexp.MustCompile(`(?i)[/\s]*@tag\("binding"[,\s"]+([^"]+)"\)\s*`)
	re2 := regexp.MustCompile(`(?i)required`)
	for _, s := range arr {
		matches := re.FindStringSubmatch(s)
		if len(matches) == 2 && re2.MatchString(matches[1]) {
			isRequired = true
		} else {
			newArr = append(newArr, s)
		}
	}
	return strings.Join(newArr, "\n"), isRequired
}

// 仅支持string和number兼容两种写法 @example=xxx 或 @example(xxx xxx)
func filterExample(doc string, ty string) (string, interface{}) {
	arr := strings.Split(doc, "\n")
	var newArr []string
	var example string
	re := regexp.MustCompile(`(?i)\s*//\s*@example=(.*|\("*[^)]+"*)`)
	re2 := regexp.MustCompile(`(?i)[/\s]*@example\((.*)\)\s*`)
	for _, s := range arr {
		matches := re.FindStringSubmatch(s)
		if len(matches) == 2 {
			example = matches[1]
		} else {
			matches = re2.FindStringSubmatch(s)
			if len(matches) == 2 {
				example = matches[1]
			} else {
				newArr = append(newArr, s)
			}
		}
	}
	var result interface{}
	if example != "" {
		example = strings.Trim(strings.Trim(example, "\""), " ")
		t := getProtoToSwagger(ty)
		switch t {
		case "string":
			result = example
		case "integer", "number":
			result, _ = strconv.ParseFloat(example, 64)
		case "boolean":
			result, _ = strconv.ParseBool(example)
		}
	}
	return strings.Join(newArr, "\n"), result
}

// 首行为默认说明，次行如：//@lang=zh xxx 中的 xxx 将替换默认说明
func filterLanguage(doc string) string {
	arr := strings.Split(doc, "\n")
	if len(arr) < 2 {
		return doc
	}
	var newArr []string
	for i, s := range arr {
		if i != 0 {
			re := regexp.MustCompile(`(?i)\s*//\s*@lang=([a-z]+)\s*(.*)`)
			match := re.FindStringSubmatch(s)
			if len(match) == 3 {
				if language == match[1] {
					newArr[0] = match[2]
				}
				continue
			}
		}
		newArr = append(newArr, s)
	}
	return strings.Join(newArr, "\n")
}

func filterTag(str string) string {
	arr := strings.Split(str, "\n")
	var newArr []string
	re := regexp.MustCompile(`(?i)\s*//\s*@tag\("([a-zA-Z]+)"[,\s"]+"([^"]+)"\)`)
	for _, s := range arr {
		if !re.MatchString(s) {
			newArr = append(newArr, s)
		}
	}
	return strings.Join(newArr, "\n")
}

func parseTag(str string) string {
	arr := strings.Split(str, "\n")
	var newArr []string
	re := regexp.MustCompile(`(?i)\s*//\s*@tag\("([a-zA-Z]+)"[,\s"]+"([^"]+)"\)`)
	for _, s := range arr {
		match := re.FindStringSubmatch(s)
		if len(match) == 3 {
			newArr = append(newArr, "`"+match[1]+": "+match[2]+"`")
		} else {
			newArr = append(newArr, s)
		}
	}
	return strings.Join(newArr, "  \n")
}

// 例：@query=id @lang=zh @format=string @example=abc 用户ID
// format默认为int，如format是对象或枚举，即使本地引用，也必须要加上包名，如@format=api.UserStatus
func parseParamInPath(option parser.Option) (params openapi.Parameters) {
	re := regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)`)
	matches := re.FindAllStringSubmatch(option.Val, -1)
	for _, match := range matches {
		key := match[1]
		var doc, example string
		format := "int" //默认int（int非protobuf类型）
		if option.Doc != "" {
			var correctLang, lockDoc bool
			for _, s := range strings.Split(option.Doc, "\n") {
				r := regexp.MustCompile(`(?i)@([a-z_]*)=([a-z0-9_.]*)`)
				ms := r.FindAllStringSubmatch(s, -1)
				correctDoc := false
				for _, m := range ms {
					switch m[1] {
					case "query":
						if m[2] == key {
							correctDoc = true
						}
					case "lang":
						if m[2] == language {
							correctLang = true
						} else {
							correctDoc = false
						}
					case "format":
						if correctDoc {
							format = m[2]
						}
					case "example":
						if correctDoc {
							example = m[2]
						}
					}
				}
				if correctDoc && !lockDoc {
					doc = filterTag(strings.Trim(strings.Trim(r.ReplaceAllString(s, ""), "/"), " "))
					if correctLang {
						lockDoc = true
					}
				}
			}
		}
		p := &openapi.Parameter{
			Name:        key,
			Description: doc,
			Format:      format,
			In:          "path",
			Required:    true,
			Type:        getProtoToSwagger(format),
			Example:     example,
		}
		if p.Type == "object" {
			p.Schema = &openapi.Schema{
				Ref: "#/definitions/" + format,
			}
		}
		params = append(params, p)
	}
	return
}
