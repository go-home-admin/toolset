package commands

import (
	"bytes"
	"fmt"
	"github.com/ctfang/command"
	"github.com/go-home-admin/toolset/parser"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// RouteCommand @Bean
type RouteCommand struct{}

// 简单gin处理模版
var goMethodStr = `package {package}

import ({import})

// {action} {doc}
func (receiver *Controller) {action}(req *{paramAlias}.{param}, ctx http.Context) (*{returnAlias}.{return}, error) {
	// TODO 这里写业务
	return &{returnAlias}.{return}{}, nil
}

// GinHandle{action} gin原始路由处理
// http.{method}({url})
func (receiver *Controller) GinHandle{action}(ctx *gin.Context) {
	req := &{paramAlias}.{param}{}
	err := ctx.ShouldBind(req)
	context := http.NewContext(ctx)
	if err != nil {
		context.Fail(err)
		return
	}
	
	resp, err := receiver.{action}(req, context)
	if err != nil {
		context.Fail(err)
		return
	}

	context.Success(resp)
}
`

// http同时调用到grpc入口
var goMethodStrForCallGrpc = `package {package}

import ({import})

// {action}  {doc}
func (receiver *Controller) {action}(req *{paramAlias}.{param}, ctx http.Context) (*{returnAlias}.{return}, error) {
	return myGrpc.NewHandle().{action}(context.Background(), req)
}

// GinHandle{action} gin原始路由处理
// http.{method}({url})
func (receiver *Controller) GinHandle{action}(ctx *gin.Context) {
	req := &{paramAlias}.{param}{}
	err := ctx.ShouldBind(req)
	context := http.NewContext(ctx)
	if err != nil {
		context.Fail(err)
		return
	}
	
	resp, err := receiver.{action}(req, context)
	if err != nil {
		context.Fail(err)
		return
	}

	context.Success(resp)
}
`

// grpc入口
var goGrpcFunc = `package {package}

import ({import})

// {action}{doc}
// http.{method}({url})
func (h Handle) {action}(ctx context.Context, req *{paramAlias}.{param}) (*{returnAlias}.{return}, error) {
	// TODO 这里写业务
	return &{returnAlias}.{return}{}, nil
}
`

var goConStr = `package {package}

// Controller @Bean
type Controller struct {
}`

var goGrpcStr = `package {package}

// Handle @Bean
type Handle struct {
}`

func (RouteCommand) Configure() command.Configure {
	return command.Configure{
		Name:        "make:route",
		Description: "根据protoc文件定义, 生成路由信息和控制器文件",
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
					Default:     "@root/routes",
				},
				{
					Name:        "out_actions",
					Description: "生成文件到指定目录",
					Default:     "@root/app/http",
				},
				{
					Name:        "skip",
					Description: "跳过某个目录, 不生成api信息",
					Default:     "", // @root/protobuf/http
				},
			},
			Has: []command.ArgParam{
				{
					Name:        "--grpc",
					Description: "生成grpc文件处理模版",
				},
			},
		},
	}
}

func repRootPath(input command.Input) command.Input {
	root := getRootPath()

	for str, li := range input.Option {
		for i, s := range li {
			li[i] = strings.Replace(s, "@root", root, 1)
		}
		input.Option[str] = li
	}

	return input
}

func (RouteCommand) Execute(input command.Input) {
	input = repRootPath(input)
	module := getModModule()
	out := input.GetOption("out_route")
	outHttp := input.GetOption("out_actions")
	path := input.GetOption("path")
	skips := input.GetOptions("skip")

	agl := map[string]*ApiGroups{}

	for dir, parsers := range parser.NewProtocParserForDir(path) {
		for _, skip := range skips {
			if strings.Index(dir, skip) != -1 {
				fmt.Println("skip dir = " + dir)
				continue
			}
		}
		for _, fileParser := range parsers {
			for _, service := range fileParser.Services {
				group := ""

				for _, option := range service.Opt {
					if option.Key == "http.RouteGroup" {
						group = option.Val
						if _, ok := agl[group]; !ok {
							agl[group] = &ApiGroups{
								name: group,
								imports: map[string]string{
									"home_api_1": "github.com/go-home-admin/home/bootstrap/http/api",
									"home_gin_1": "github.com/gin-gonic/gin",
								},
								controllers: make([]Controller, 0),
								servers:     make([]parser.Service, 0),
							}
						}
						genController(service, outHttp, input.GetHas("--grpc"))
						break
					}
				}

				if group != "" {
					g := agl[group]
					imports := strings.Replace(outHttp, getRootPath(), module, 1) + "/" + fileParser.PackageName + "/" + parser.StringToSnake(service.Name)
					g.imports[imports] = imports

					g.controllers = append(g.controllers, Controller{
						name:  service.Name,
						alias: imports,
					})

					g.servers = append(g.servers, service)
				}
			}
		}
	}

	_ = os.RemoveAll(out)
	_ = os.MkdirAll(out, 0765)

	for _, g := range agl {
		genRoute(g, out)
	}
	cmd := exec.Command("go", []string{"fmt", out}...)
	var outBuffer bytes.Buffer
	cmd.Stdout = &outBuffer
	cmd.Stderr = os.Stderr
	cmd.Dir = out
	_ = cmd.Run()
}

func genController(server parser.Service, out string, genGrpc bool) {
	page := server.Protoc.PackageName
	out += "/" + page + "/" + parser.StringToSnake(server.Name)

	if !parser.DirIsExist(out) {
		_ = os.MkdirAll(out, 0760)
	}

	for s, _ := range server.Opt {
		if s == "http.Resource" {
			genResourceController(server, out)
		}
	}

	if !parser.DirIsExist(out + "/controller.go") {
		conStr := goConStr
		conStr = strings.ReplaceAll(conStr, "{package}", parser.StringToSnake(server.Name))
		err := os.WriteFile(out+"/controller.go", []byte(conStr), 0760)
		if err != nil {
			panic(err)
		}
	}

	gin := "github.com/gin-gonic/gin"
	http := "github.com/go-home-admin/home/app/http"
	imports := map[string]string{gin: gin, http: http}
	goMod := getModModule()
	methodStr := goMethodStr

	grpcOut := strings.ReplaceAll(out, "/app/http", "/app/grpc")
	if genGrpc {
		methodStr = goMethodStrForCallGrpc
		imports["myGrpc"] = goMod + "/app/grpc/" + parser.StringToSnake(server.Protoc.PackageName) + "/" + parser.StringToSnake(server.Name)
		imports["context"] = "context"
		// 生成grpc handle
		genGrpcHandle(grpcOut, server)
	}
	for rName, rpc := range server.Rpc {
		for _, options := range rpc.Opt {
			for _, option := range options {
				if strings.Index(option.Key, "http.") == 0 {
					actionName := parser.StringToHump(rName)
					if parser.DirIsExist(out+"/"+parser.StringToSnake(actionName)+"_action.go") &&
						(parser.DirIsExist(grpcOut+"/"+parser.StringToSnake(actionName)+"_action.go") || !genGrpc) {
						continue
					}

					serPage := goMod + "/generate/proto/" + server.Protoc.PackageName
					imports[serPage] = serPage
					importsStr := ""
					m := genImportAlias(imports)
					sk := sortMap(m)
					for _, s := range sk {
						importsStr += "\n\t" + m[s] + " \"" + s + "\""
					}

					controllerAlias := m[serPage]
					paramAlias := controllerAlias
					params := rpc.Param
					returnAlias := controllerAlias
					ret := rpc.Return

					if i := strings.Index(rpc.Param, "."); i != -1 {
						paramAlias = rpc.Param[:i]
						params = rpc.Param[i+1:]
						log.Printf("%v 不是同级目录的包, 生成代码后需要手动加入\n", rpc.Param)
					}
					if i := strings.Index(rpc.Return, "."); i != -1 {
						returnAlias = rpc.Return[:i]
						ret = rpc.Return[i+1:]
						log.Printf("%v 不是同级目录的包, 生成代码后需要手动加入\n", rpc.Return)
					}
					// 包没有被使用
					if params != rpc.Param && ret != rpc.Return {
						delete(imports, serPage)
					}

					str := methodStr
					i := strings.Index(option.Key, ".")
					method := option.Key[i+1:]
					url := option.Val

					reps := map[string]string{
						"{package}":         parser.StringToSnake(server.Name),
						"{import}":          importsStr + "\n",
						"{doc}":             rpc.Doc,
						"{method}":          method,
						"{url}":             url,
						"{action}":          actionName,
						"{controllerAlias}": controllerAlias,
						"{paramAlias}":      paramAlias,
						"{param}":           params,
						"{returnAlias}":     returnAlias,
						"{return}":          ret,
					}

					// 生成http
					if !parser.DirIsExist(out + "/" + parser.StringToSnake(actionName) + "_action.go") {
						for s, O := range reps {
							str = strings.ReplaceAll(str, s, O)
						}
						err := os.WriteFile(out+"/"+parser.StringToSnake(actionName)+"_action.go", []byte(str), 0766)
						if err != nil {
							log.Fatal(err)
						}
					}

					// 生成grpc
					if genGrpc && !parser.DirIsExist(grpcOut+"/"+parser.StringToSnake(actionName)+"_action.go") {
						reps["{import}"] = "\n\t\"context\"\n\t\"" + serPage + "\"\n"

						goGrpcFunc2 := goGrpcFunc
						for s, O := range reps {
							goGrpcFunc2 = strings.ReplaceAll(goGrpcFunc2, s, O)
						}
						err := os.WriteFile(grpcOut+"/"+parser.StringToSnake(actionName)+"_action.go", []byte(goGrpcFunc2), 0766)
						if err != nil {
							log.Fatal(err)
						}
					}
				}
			}
		}
	}
}

func genGrpcHandle(out string, server parser.Service) {
	if !parser.DirIsExist(out) {
		_ = os.MkdirAll(out, 0760)
	}

	if !parser.DirIsExist(out + "/handle.go") {
		conStr := strings.ReplaceAll(goGrpcStr, "{package}", parser.StringToSnake(server.Name))
		err := os.WriteFile(out+"/handle.go", []byte(conStr), 0760)
		if err != nil {
			panic(err)
		}
	}
}

// 生成资源控制器
func genResourceController(server parser.Service, out string) {
	actionFile := out + "/crud.go"
	if parser.DirIsExist(actionFile) {
		return
	}

	str := `package {package}

import (
	"github.com/gin-gonic/gin"
	"github.com/go-home-admin/amis"
)

func (c *CrudContext) Common() {
	// c.SetDb(admin.NewOrmAdminMenu())
}

func (c *CrudContext) Table(curd *amis.Crud) {
	curd.Column("自增", "id")
	curd.Column("文本", "text")
	curd.Column("图片", "image").Image()
	curd.Column("日期", "date").Date()
	curd.Column("进度", "progress").Progress()
	curd.Column("状态", "status").Status()
	curd.Column("开关", "switch").Switch()
	curd.Column("映射", "mapping").Mapping(map[string]string{})
	curd.Column("List", "list").List()
}

func (c *CrudContext) Form(form *amis.Form) {
	form.Input("text", "文本")
	form.Input("image", "图片")

}

func (c *Controller) GinHandleCurd(ctx *gin.Context) {
	var crud = &CrudContext{}
	crud.CurdController.Context = ctx
	crud.CurdController.Crud = crud
	amis.GinHandleCurd(ctx, crud)
}

type CrudContext struct {
	amis.CurdController
}

`
	str = strings.ReplaceAll(str, "{package}", parser.StringToSnake(server.Name))
	err := os.WriteFile(actionFile, []byte(str), 0766)
	if err != nil {
		log.Fatal(err)
	}
}

func genRoute(g *ApiGroups, out string) {
	context := make([]string, 0)
	context = append(context, "package routes")

	// import
	importAlias := genImportAlias(g.imports)
	if len(importAlias) != 0 {
		context = append(context, "\nimport ("+parser.GetImportStrForMap(importAlias)+"\n)")
	}
	// Routes struct
	context = append(context, genRoutesStruct(g, importAlias))
	// Routes struct func GetRoutes
	context = append(context, genRoutesFunc(g, importAlias))

	str := "// gen for home toolset"
	for _, s2 := range context {
		str = str + "\n" + s2
	}
	err := os.WriteFile(out+"/"+parser.StringToSnake(g.name)+"_route.go", []byte(str), 0766)
	if err != nil {
		log.Fatal("无法写入目录文件", err)
	}
}

func genRoutesFunc(g *ApiGroups, m map[string]string) string {
	homeGin := m["github.com/gin-gonic/gin"]
	homeApi := m["github.com/go-home-admin/home/bootstrap/http/api"]

	str := "func (c *" + parser.StringToHump(g.name) + "Routes) GetGroup() string {" +
		"\n\treturn \"" + g.name + "\"" +
		"\n}"

	str += "\nfunc (c *" + parser.StringToHump(g.name) + "Routes) GetRoutes() map[*" + homeApi + ".Config]func(c *" + homeGin + ".Context) {" +
		"\n\treturn map[*" + homeApi + ".Config]func(c *" + homeGin + ".Context){"

	for _, server := range g.servers {
		for _, s := range forOpt(server.Opt) {
			v := server.Opt[s]
			if s == "http.Resource" {
				str += "\n\t\t" + homeApi + ".Get(\"" + v.Val + "\"):" + "c." + parser.StringToSnake(server.Name) + ".GinHandleCurd,"
				str += "\n\t\t" + homeApi + ".Post(\"" + v.Val + "\"):" + "c." + parser.StringToSnake(server.Name) + ".GinHandleCurd,"
				str += "\n\t\t" + homeApi + ".Any(\"" + v.Val + "/:action\"):" + "c." + parser.StringToSnake(server.Name) + ".GinHandleCurd,"
			}
		}
		for _, rName := range forServerOpt(server.Rpc) {
			rpc := server.Rpc[rName]
			for _, options := range rpc.Opt {
				for _, option := range options {
					if strings.Index(option.Key, "http.") == 0 {
						i := strings.Index(option.Key, ".")
						method := option.Key[i+1:]
						str += "\n\t\t" + homeApi + "." + method + "(\"" + option.Val + "\"):" +
							"c." + parser.StringToSnake(server.Name) + ".GinHandle" + parser.StringToHump(rName) + ","
					}
				}
			}
		}
	}

	return str + "\n\t}\n}"
}

func forServerOpt(m map[string]parser.ServiceRpc) []string {
	keys := make([]string, 0)
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

func forOpt(m map[string]parser.Option) []string {
	keys := make([]string, 0)
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

func genRoutesStruct(g *ApiGroups, m map[string]string) string {
	str := "\n// @Bean" +
		"\ntype " + parser.StringToHump(g.name) + "Routes struct {\n"
	for _, controller := range g.controllers {
		alias := m[controller.alias]
		str += "\t" + parser.StringToSnake(controller.name) + " *" + alias + ".Controller" + " `inject:\"\"`\n"
	}

	return str + "}\n"
}

type ApiGroups struct {
	name        string
	imports     map[string]string
	controllers []Controller
	servers     []parser.Service
}

type Controller struct {
	name  string
	alias string
	ty    string // *alias.Controller
}
