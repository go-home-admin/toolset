package commands

import (
	"encoding/json"
	"fmt"
	"github.com/ctfang/command"
	"github.com/go-home-admin/home/bootstrap/utils"
	"github.com/go-home-admin/toolset/console/commands/openapi"
	"github.com/go-home-admin/toolset/parser"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
)

// Js @Bean
type Js struct{}

var isResponse bool

func (j *Js) Configure() command.Configure {
	return command.Configure{
		Name:        "make:js",
		Description: "根据swagger生成js请求文件",
		Input: command.Argument{
			Option: []command.ArgParam{
				{
					Name:        "in",
					Description: "swagger.json路径, 可本地可远程",
					Default:     "@root/web/swagger.json",
				},
				{
					Name:        "out",
					Description: "js文件输出路径",
					Default:     "@root/resources/src/api/swagger_gen.js",
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

func (j *Js) Execute(input command.Input) {
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

	swagger := openapi.Spec{
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
	_ = json.Unmarshal([]byte(inSwaggerStr), &swagger)
	fixSwaggerType(&swagger)

	tag := input.GetOption("tag")
	infoTags := input.GetOptions("info_tags")
	str := fmt.Sprintf("import http from '%s'\n", input.GetOption("http_from"))
	for _, url := range sortPathMap(swagger.Paths) {
		paths := swagger.Paths[url]
		re, _ := regexp.Compile("\\$\\[.+\\]")
		url = re.ReplaceAllString(url, "")
		url, funcName, params := analysisUrl(url)
		urlParams := make([]*openapi.Parameter, 0)
		for _, p := range params {
			urlParams = append(urlParams, &openapi.Parameter{
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
			if method.cm {
				isResponse = false
				//Tags说明
				var tagInfo string
				for _, s := range infoTags {
					info := getTagInfo(method.e.Description, s)
					if info != "" {
						tagInfo += fmt.Sprintf("\n * @%s %s", s, info)
					}
				}
				var paramNames []string
				paramStr := genJsRequest(method.e.Parameters, swagger)
				var dataStr string
				if paramStr != "" {
					paramNames = append(paramNames, "data")
					dataStr = ", data"
				}
				if len(urlParams) > 0 {
					for _, urlParam := range urlParams {
						paramStr += "\n * @param {string|number} " + urlParam.Name
						paramNames = append(paramNames, urlParam.Name)
					}
				}
				var response string
				if _, ok := method.e.Responses["200"]; ok {
					if method.e.Responses["200"].Schema != nil {
						isResponse = true
						response = getObjectStrFromRef(method.e.Responses["200"].Schema.Ref, swagger)
					}
				}
				str += fmt.Sprintf(`
/**
 * %v%v%v
 * @returns {Promise<{code:number,data:%v,message:string}>}
 * @callback
 */
export async function %v%v(%v) {
  return await http.%v(%v%v)
}
`,
					method.e.Summary,
					tagInfo,
					paramStr,
					response,
					parser.StringToHump(strings.Trim(strings.ReplaceAll(funcName, "/", "_"), "_")),
					parser.StringToHump(method.method),
					strings.Join(paramNames, ", "),
					method.method,
					url,
					dataStr,
				)
			}
		}
	}
	_ = os.WriteFile(out, []byte(str), 0766)
}

func sortPathMap(m map[string]*openapi.Path) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	//按字典升序排列
	sort.Strings(keys)
	return keys
}

func genJsRequest(p openapi.Parameters, swagger openapi.Spec) string {
	if len(p) == 0 {
		return ""
	}
	str := "\n * @param {{"
	for i, parameter := range p {
		t := parameter.Type
		switch parameter.Type {
		case "integer", "Number":
			t = "number"
		case "array":
			t = "[]"
			if parameter.Format != "" {
				if ty, ok := protoToSwagger[parameter.Format]; ok {
					parameter.Format = ty
				}
				t = parameter.Format + t
			} else if parameter.Items != nil {
				t = getObjectStrFromRef(parameter.Items.Ref, swagger) + t
			}
		case "", "object":
			if parameter.Schema != nil {
				t = getObjectStrFromRef(parameter.Schema.Ref, swagger)
			}
		}
		if i != 0 {
			str += ","
		}
		if !parameter.Required {
			parameter.Name = parameter.Name + "?"
		}
		str += fmt.Sprintf(`%v:%v`, parameter.Name, t)
	}
	return str + "}} data"
}

type makeJsCache struct {
	e      *openapi.Endpoint
	cm     bool
	method string
}

func canMakeJs(e *openapi.Endpoint, tag string) bool {
	makeJs := false
	if e != nil {
		if tag == "" {
			makeJs = true
		} else {
			for _, t := range e.Tags {
				if t == tag {
					makeJs = true
					break
				}
			}
		}
	}

	return makeJs
}

func analysisUrl(url string) (newUrl string, funcName string, params []string) {
	re, _ := regexp.Compile("/:([^/\\n\\r])+")
	funcName = url
	newUrl = fmt.Sprintf("`%v`", url)
	for _, s := range re.FindAllString(funcName, -1) {
		p := strings.Replace(s, "/:", "", 1)
		params = append(params, p)
		funcName = strings.Replace(funcName, s, "_"+p, 1)
		newUrl = strings.Replace(newUrl, s, "/${"+p+"}", 1)
	}
	return
}

func fixSwaggerType(swagger *openapi.Spec) {
	for url, path := range swagger.Paths {
		if path.Get != nil {
			for i, parameter := range path.Get.Parameters {
				if parameter.Schema != nil {
					key := strings.Replace(parameter.Schema.Ref, "#/definitions/", "", 1)
					if _, ok := swagger.Definitions[key]; ok {
						swagger.Paths[url].Get.Parameters[i].Type = swagger.Definitions[key].Format
					}
				}
				swagger.Paths[url].Get.Parameters[i].Type = strings.ToLower(swagger.Paths[url].Get.Parameters[i].Type)
			}
		}
		if path.Post != nil {
			for i, parameter := range path.Post.Parameters {
				if parameter.Schema != nil {
					key := strings.Replace(parameter.Schema.Ref, "#/definitions/", "", 1)
					if _, ok := swagger.Definitions[key]; ok {
						swagger.Paths[url].Post.Parameters[i].Type = swagger.Definitions[key].Format
					}
				}
				swagger.Paths[url].Post.Parameters[i].Type = strings.ToLower(swagger.Paths[url].Post.Parameters[i].Type)
			}
		}
		if path.Put != nil {
			for i, parameter := range path.Put.Parameters {
				if parameter.Schema != nil {
					key := strings.Replace(parameter.Schema.Ref, "#/definitions/", "", 1)
					if _, ok := swagger.Definitions[key]; ok {
						swagger.Paths[url].Put.Parameters[i].Type = swagger.Definitions[key].Format
					}
				}
				swagger.Paths[url].Put.Parameters[i].Type = strings.ToLower(swagger.Paths[url].Put.Parameters[i].Type)
			}
		}
		if path.Patch != nil {
			for i, parameter := range path.Patch.Parameters {
				if parameter.Schema != nil {
					key := strings.Replace(parameter.Schema.Ref, "#/definitions/", "", 1)
					if _, ok := swagger.Definitions[key]; ok {
						swagger.Paths[url].Patch.Parameters[i].Type = swagger.Definitions[key].Format
					}
				}
				swagger.Paths[url].Patch.Parameters[i].Type = strings.ToLower(swagger.Paths[url].Patch.Parameters[i].Type)
			}
		}
		if path.Delete != nil {
			for i, parameter := range path.Delete.Parameters {
				if parameter.Schema != nil {
					key := strings.Replace(parameter.Schema.Ref, "#/definitions/", "", 1)
					if _, ok := swagger.Definitions[key]; ok {
						swagger.Paths[url].Delete.Parameters[i].Type = swagger.Definitions[key].Format
					}
				}
			}
		}
	}
}

func getObjectStrFromRef(ref string, swagger openapi.Spec) string {
	str := "{"
	def := strings.Replace(ref, "#/definitions/", "", 1)
	var params []string
	if _, ok := swagger.Definitions[def]; ok {
		if isEnum(swagger.Definitions[def]) {
			return "number"
		}
		for key, schema := range swagger.Definitions[def].Properties {
			if key == "list" {
				utils.Dump(key)
			}
			if !isResponse && !parser.InArrString(key, swagger.Definitions[def].Required) {
				key = key + "?"
			}
			params = append(params, fmt.Sprintf(`%v:%v`, key, getJsType(schema, swagger, ref)))
		}
	}
	str += strings.Join(params, ",")
	str += "}"
	return str
}

func getJsType(schema *openapi.Schema, swagger openapi.Spec, ref string) string {
	t := schema.Type
	switch schema.Type {
	case "integer", "Number":
		t = "number"
	case "array":
		if schema.Items != nil {
			t = getJsType(schema.Items, swagger, ref)
		}
		t += "[]"
	case "object", "":
		if ref == schema.Ref {
			t = "{}"
		} else if schema.Ref != "" {
			t = getObjectStrFromRef(schema.Ref, swagger)
		}
	}
	return t
}
