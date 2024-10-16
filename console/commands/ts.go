package commands

import (
	"encoding/json"
	"fmt"
	"github.com/ctfang/command"
	"github.com/go-home-admin/toolset/console/commands/openapi"
	"github.com/go-home-admin/toolset/parser"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Ts @Bean
type Ts struct {
	enums   map[string]string
	params  map[string]string
	objects map[string]string
	swagger openapi.Spec
}

func (t *Ts) Init() {
	t.enums = make(map[string]string)
	t.params = make(map[string]string)
	t.objects = make(map[string]string)
}

func (t *Ts) Configure() command.Configure {
	return command.Configure{
		Name:        "make:ts",
		Description: "根据swagger生成ts结构及接口请求方法",
		Input: command.Argument{
			Option: []command.ArgParam{
				{
					Name:        "in",
					Description: "swagger.json路径, 可本地可远程",
					Default:     "@root/web/swagger_gen.json",
				},
				{
					Name:        "out",
					Description: "ts文件输出路径",
					Default:     "@root/resources/src/api/swagger_gen.ts",
				},
				{
					Name:        "tag",
					Description: "只生成指定tag的请求",
				},
				{
					Name:        "http_from",
					Description: "指定import的http函数位置",
					Default:     "@/utils/request",
				},
				{
					Name:        "info_tags",
					Description: "指定注释中的tag显示于接口说明",
				},
			},
		},
	}
}

func (t *Ts) Execute(input command.Input) {
	root := getRootPath()
	in := input.GetOption("in")
	in = strings.Replace(in, "@root", root, 1)
	inSwaggerStr := ""
	if strings.Index(in, "http") == 0 {
		// 远程获取文件
		req, _ := http.NewRequest("GET", in, nil)
		res, _ := http.DefaultClient.Do(req)
		defer res.Body.Close()
		//得到返回结果
		body, _ := io.ReadAll(res.Body)
		inSwaggerStr = string(body)
	} else {
		body, _ := os.ReadFile(in)
		inSwaggerStr = string(body)
	}
	out := input.GetOption("out")
	out = strings.Replace(out, "@root", root, 1)

	t.swagger = openapi.Spec{
		Swagger: "2.0",
		Info: openapi.Info{
			Title:       "2",
			Description: "2",
			Version:     "2",
		},
		Host:     "api.swagger.com",
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
	_ = json.Unmarshal([]byte(inSwaggerStr), &t.swagger)
	fixSwaggerType(&t.swagger)

	tag := input.GetOption("tag")
	infoTags := input.GetOptions("info_tags")
	str := fmt.Sprintf("import http from '%s'\n", input.GetOption("http_from"))
	for _, url := range sortPathMap(t.swagger.Paths) {
		paths := t.swagger.Paths[url]
		re, _ := regexp.Compile("\\$\\[.+\\]")
		url = re.ReplaceAllString(url, "")
		url, funcName, params := analysisUrl(url)
		urlQuery := make([]*openapi.Parameter, 0)
		for _, p := range params {
			urlQuery = append(urlQuery, &openapi.Parameter{
				Name:     p,
				Required: true,
				Type:     "string",
			})
		}
		methods := make([]makeJsCache, 0)
		methods = append(methods, makeJsCache{e: paths.Get, cm: canMakeJs(paths.Get, tag), method: "get"})
		methods = append(methods, makeJsCache{e: paths.Put, cm: canMakeJs(paths.Put, tag), method: "put"})
		methods = append(methods, makeJsCache{e: paths.Post, cm: canMakeJs(paths.Post, tag), method: "post"})
		methods = append(methods, makeJsCache{e: paths.Patch, cm: canMakeJs(paths.Patch, tag), method: "patch"})
		methods = append(methods, makeJsCache{e: paths.Delete, cm: canMakeJs(paths.Delete, tag), method: "delete"})
		for _, method := range methods {
			if !method.cm {
				continue
			}
			//Tags说明
			var tagInfo string
			for _, s := range infoTags {
				info := getTagInfo(method.e.Description, s)
				if info != "" {
					tagInfo += fmt.Sprintf("\n * @%s %s", s, info)
				}
			}
			//func名
			fName := parser.StringToHump(strings.Trim(strings.ReplaceAll(funcName, "/", "_"), "_"))
			//请求参数
			var paramStr, hasParams string
			if method.method == "get" && len(method.e.Parameters) > 0 {
				typeName := fName + parser.StringToHump(method.method) + "Payload"
				paramStr = "data: " + typeName
				hasParams = "\n    data,"
				t.genTsParams(typeName, method.e.Parameters)
			} else if method.e.RequestBody != nil {
				ref := strings.Replace(method.e.RequestBody.Content.Json.Schema.Ref, "#/definitions/", "", 1)
				ref = strings.ReplaceAll(ref, ".", "_")
				paramStr = "data: " + ref
				hasParams = "\n    data,"
			}
			//URL参数
			for _, urlParam := range urlQuery {
				paramStr += fmt.Sprintf(", %s: %s", urlParam.Name, "string|number")
			}
			paramStr = strings.Trim(paramStr, ", ")
			//响应结构
			var response string
			if _, ok := method.e.Responses["200"]; ok {
				if method.e.Responses["200"].Schema != nil {
					response = t.genType(method.e.Responses["200"].Schema.Ref)
				}
			}
			str += fmt.Sprintf(`
/**
 * %v%v
*/
export const %v%v = (%v) => {
  return http.%v(
    %v,%v
  ) as Promise<{code:number,data:%v,message:string}>;
}
`,
				strings.Trim(method.e.Summary, " "),
				tagInfo,
				fName,
				parser.StringToHump(method.method),
				paramStr,
				method.method,
				url,
				hasParams,
				response,
			)
		}
	}
	//插入枚举
	str += "\n"
	for _, e := range t.enums {
		str += e
	}
	//插入请求参数
	str += "\n"
	for _, e := range t.params {
		str += e
	}
	//插入结构
	str += "\n"
	for _, e := range t.objects {
		str += e
	}
	_ = os.WriteFile(out, []byte(str), 0766)
}

func (t *Ts) genTsParams(typeName string, params []*openapi.Parameter) {
	str := "export type " + typeName + " = {\n"
	for _, parameter := range params {
		ty := t.getTsTypeFromParameter(parameter)
		if !parameter.Required {
			parameter.Name = parameter.Name + "?"
		}
		if parameter.Description != "" {
			str += fmt.Sprintf("  // %s\n", t.clearEmpty(parameter.Description))
		}
		str += fmt.Sprintf("  %v: %v;\n", parameter.Name, ty)
	}
	str += "}\n"
	t.params[typeName] = str
}

func (t *Ts) genType(ref string) string {
	def := strings.Replace(ref, "#/definitions/", "", 1)
	key := strings.ReplaceAll(def, ".", "_")
	if _, ok := t.objects[key]; ok {
		return key
	}
	if _, ok := t.enums[key]; ok {
		return key
	}
	if _, ok := t.swagger.Definitions[def]; ok {
		if isEnum(t.swagger.Definitions[def]) {
			t.genEnums(key, t.swagger.Definitions[def])
			return key
		}
		if len(t.swagger.Definitions[def].Properties) == 0 {
			return "{}"
		}
		str := fmt.Sprintf("export type %s = {\n", key)
		for k, schema := range t.swagger.Definitions[def].Properties {
			if schema.Description != "" {
				str += fmt.Sprintf("  // %s\n", t.clearEmpty(schema.Description))
			} else if len(schema.AllOf) > 0 {
				for _, s := range schema.AllOf {
					if s.Description != "" {
						str += fmt.Sprintf("  // %s\n", t.clearEmpty(s.Description))
					}
				}
			}
			str += fmt.Sprintf("  %s: %s;\n", k, t.getTsTypeFromSchema(schema, ref))
		}
		str += "}\n"
		t.objects[key] = str
	}
	return key
}

func (t *Ts) getTsTypeFromParameter(param *openapi.Parameter) string {
	if param.Schema != nil {
		return t.getTsTypeFromSchema(param.Schema, param.Format)
	}
	return t.getTsTypeFromSchema(&openapi.Schema{
		Description: param.Description,
		Ref:         param.Ref,
		Type:        param.Type,
		Format:      param.Format,
		Enum:        param.Enum,
		Items:       param.Items,
	}, "")
}

func (t *Ts) getTsTypeFromSchema(schema *openapi.Schema, ref string) string {
	ty := schema.Type
	_ref := schema.Ref
	if len(schema.AllOf) > 0 {
		for _, s := range schema.AllOf {
			if s.Ref != "" {
				_ref = s.Ref
			}
		}
	}
	switch schema.Type {
	case "integer", "Number":
		ty = "number"
	case "array":
		if schema.Items != nil {
			ty = t.getTsTypeFromSchema(schema.Items, ref)
		}
		ty += "[]"
	case "", "object":
		if ref == _ref {
			//结构引用自己，防止死循环
			ty = strings.ReplaceAll(strings.Replace(ref, "#/definitions/", "", 1), ".", "_")
		} else if _ref != "" {
			ty = t.genType(_ref)
		}
	}
	return ty
}

func (t *Ts) genEnums(enumName string, schema *openapi.Schema) {
	str := fmt.Sprintf("export enum %s {\n", enumName)
	var keys []int
	for k, _ := range schema.Properties {
		i, _ := strconv.Atoi(k)
		keys = append(keys, i)
	}
	sort.Ints(keys)
	for _, i := range keys {
		k := strconv.Itoa(i)
		desc := t.clearEmpty(strings.TrimLeft(schema.Properties[k].Description, "enum|"))
		if desc != "" {
			str += fmt.Sprintf("  // %s\n", desc)
		}
		str += fmt.Sprintf("  %s = %s,\n", schema.Properties[k].Type, k)
	}
	str += "}\n"
	t.enums[enumName] = str
}

func (t *Ts) clearEmpty(str string) string {
	return strings.Trim(strings.ReplaceAll(str, "\n", ""), " ")
}

func isEnum(schema *openapi.Schema) bool {
	if first, ok := schema.Properties["0"]; ok && strings.Index(first.Description, "enum") == 0 {
		return true
	}
	return false
}

func getTagInfo(doc, tag string) string {
	if tag == "" {
		return ""
	}
	arr := strings.Split(doc, "\n")
	re := regexp.MustCompile("`" + tag + ":([^`]+)")
	for _, s := range arr {
		matches := re.FindStringSubmatch(s)
		if matches != nil {
			return strings.Trim(matches[1], " ")
		}
	}
	return ""
}
