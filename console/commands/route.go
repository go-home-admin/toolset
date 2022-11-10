package commands

import (
	"bytes"
	"github.com/ctfang/command"
	"github.com/go-home-admin/toolset/parser"
	"log"
	"os"
	"os/exec"
	"strings"
)

// RouteCommand @Bean
type RouteCommand struct{}

var goMethodStr = `package {package}

import ({import})

// {action}  {doc}
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

var goConStr = `package {package}

// Controller @Bean
type Controller struct {
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
			},
		},
	}
}

func (RouteCommand) Execute(input command.Input) {
	root := getRootPath()
	module := getModModule()
	out := input.GetOption("out_route")
	out = strings.Replace(out, "@root", root, 1)

	outHttp := input.GetOption("out_actions")
	outHttp = strings.Replace(outHttp, "@root", root, 1)

	path := input.GetOption("path")
	path = strings.Replace(path, "@root", root, 1)

	agl := map[string]*ApiGroups{}

	for _, parsers := range parser.NewProtocParserForDir(path) {
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
						genController(service, outHttp)
						break
					}
				}

				if group != "" {
					g := agl[group]
					imports := module + "/app/http/" + fileParser.PackageName + "/" + parser.StringToSnake(service.Name)
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

func genController(server parser.Service, out string) {
	page := server.Protoc.PackageName
	out += "/" + page + "/" + parser.StringToSnake(server.Name)

	if !parser.DirIsExist(out) {
		_ = os.MkdirAll(out, 0760)
	}

	if !parser.DirIsExist(out + "/controller.go") {
		conStr := goConStr
		conStr = strings.ReplaceAll(conStr, "{package}", parser.StringToSnake(server.Name))
		err := os.WriteFile(out+"/controller.go", []byte(conStr), 0760)
		if err != nil {
			panic(err)
		}
	}

	for s, _ := range server.Opt {
		if s == "http.Resource" {
			genResourceController(server, out)
		}
	}

	methodStr := goMethodStr
	gin := "github.com/gin-gonic/gin"
	http := "github.com/go-home-admin/home/app/http"
	imports := map[string]string{gin: gin, http: http}
	goMod := getModModule()

	for rName, rpc := range server.Rpc {
		for _, options := range rpc.Opt {
			for _, option := range options {
				if strings.Index(option.Key, "http.") == 0 {
					actionName := parser.StringToHump(rName)
					actionFile := out + "/" + parser.StringToSnake(actionName) + "_action.go"
					if parser.DirIsExist(actionFile) {
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

					for s, O := range map[string]string{
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
					} {
						str = strings.ReplaceAll(str, s, O)
					}

					err := os.WriteFile(out+"/"+parser.StringToSnake(actionName)+"_action.go", []byte(str), 0766)
					if err != nil {
						log.Fatal(err)
					}
				}
			}
		}
	}
}

// 生成资源控制器
func genResourceController(server parser.Service, out string) {
	actionFile := out + "/resource.go"
	if parser.DirIsExist(actionFile) {
		return
	}

	str := `package {package}

import (
	"github.com/gin-gonic/gin"
	"github.com/go-home-admin/go-admin/app/servers/gui"
	"github.com/go-home-admin/go-admin/app/servers/gui/form"
	"github.com/go-home-admin/go-admin/app/servers/gui/table"
)

// GinHandleResource gin原始路由处理
func (receiver *Controller) GinHandleResource(ctx *gin.Context) {
	NewGuiContext(ctx).ActionHandle()
}

type GuiContext struct {
	*gui.GinHandle
	*table.View
}

func NewGuiContext(ctx *gin.Context) *GuiContext {
	guid := &GuiContext{GinHandle: gui.NewGui(ctx)}
	guid.View = table.NewTable(guid)
	guid.SetController(guid)
	// 你要编辑的默认sql条件
	// guid.SetDb(mysql.NewOrmUser().GetDB())
	return guid
}

func (g *GuiContext) Grid(view *table.View) {
	view.Column("头像", "icon").Avatar()
	view.Column("姓名", "nickname").Width("150")
	view.Column("性别", "sex").Width("150").Filters([]gui.Filter{{Text: "男", Value: "1"}, {Text: "女", Value: "0"}})
	view.Column("邮箱", "email").Width("150")
	view.Column("注册时间", "created_at").Width("250").Sortable(true)

	action := view.NewAction()
	action.AddButton("删除").Delete().Options("type", "danger")
	action.AddButton("编辑").Edit().Options("type", "primary")

	// 设置搜索栏
	filter := view.NewSearch()
	filter.Input("name", "名称").Span("12")
	filter.Input("nick", "昵称").Span("12")
	filter.Input("sex", "性别").Span("24")

	header := view.NewHeader()
	header.Create()
}

func (g *GuiContext) Form(f *form.DialogForm) {
	f.Input("nickname", "名称")
	f.Input("created_at", "注册时间")
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
	err := os.WriteFile(out+"/"+parser.StringToSnake(g.name)+"_gen.go", []byte(str), 0766)
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
		for s, v := range server.Opt {
			if s == "http.Resource" {
				str += "\n\t\t" + homeApi + ".Get(\"" + v.Val + "\"):" + "c." + parser.StringToSnake(server.Name) + ".GinHandleResource,"
				str += "\n\t\t" + homeApi + ".Any(\"" + v.Val + "/:action\"):" + "c." + parser.StringToSnake(server.Name) + ".GinHandleResource,"
			}
		}
		for rName, rpc := range server.Rpc {
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
