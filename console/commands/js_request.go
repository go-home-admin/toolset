package commands

import (
	"encoding/json"
	"fmt"
	"github.com/ctfang/command"
	"github.com/go-home-admin/toolset/console/commands/openapi"
	"github.com/go-home-admin/toolset/parser"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strings"
)

// Js @Bean
type Js struct{}

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
					Name:        "tags",
					Description: "过滤的tag, 只生成有指定tag的请求",
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
		body, _ := ioutil.ReadAll(res.Body)
		inSwaggerStr = string(body)
	} else {
		body, _ := ioutil.ReadFile(in)
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

	tags := make(map[string]bool)
	for _, s := range input.GetOptions("tags") {
		if s != "" {
			tags[s] = true
		}
	}
	str := `
import http from "@/utils/request";
import config from "@/config";
`
	for _, url := range sortPathMap(swagger.Paths) {
		paths := swagger.Paths[url]
		methods := make([]makeJsCache, 0)
		methods = append(methods, makeJsCache{e: paths.Get, cm: canMakeJs(paths.Get, tags), method: "get"})
		methods = append(methods, makeJsCache{e: paths.Put, cm: canMakeJs(paths.Put, tags), method: "put"})
		methods = append(methods, makeJsCache{e: paths.Post, cm: canMakeJs(paths.Post, tags), method: "post"})
		methods = append(methods, makeJsCache{e: paths.Patch, cm: canMakeJs(paths.Patch, tags), method: "patch"})
		methods = append(methods, makeJsCache{e: paths.Delete, cm: canMakeJs(paths.Delete, tags), method: "delete"})
		for _, method := range methods {
			if method.cm {
				str += fmt.Sprintf(`
/**
 * %v%v
 * @returns {Promise<{code:Number,data:{},message:string}>}
 * @constructor
 */
export async function %v%v(data) {
	return await http.%v(config.API_URL + "%v", data);
}
`,
					method.e.Description,
					genJsRequest(method.e.Parameters),
					parser.StringToHump(strings.Trim(strings.ReplaceAll(url, "/", "_"), "_")),
					parser.StringToHump(method.method),
					method.method,
					url,
				)
			}
		}
	}
	fmt.Println(out)
	os.WriteFile(out, []byte(str), 0766)
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

func genJsRequest(p openapi.Parameters) string {
	if len(p) == 0 {
		return ""
	}
	str := "\n * @param {{"
	for i, parameter := range p {
		t := "{}"
		switch parameter.Type {
		case "integer":
			t = "Number"
		case "string":
			t = "string"
		}
		if i != 0 {
			str += ","
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

func canMakeJs(e *openapi.Endpoint, tags map[string]bool) bool {
	makeJs := false
	if e != nil {
		if len(tags) == 0 {
			makeJs = true
		} else {
			for _, t := range e.Tags {
				if tags[t] {
					makeJs = true
					break
				}
			}
		}
	}

	return makeJs
}
