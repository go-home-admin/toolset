package commands

import (
	"database/sql"
	"fmt"
	"github.com/ctfang/command"
	"github.com/go-home-admin/toolset/console/commands/orm"
	"github.com/go-home-admin/toolset/console/commands/pgorm"
	"github.com/go-home-admin/toolset/parser"
	"gopkg.in/yaml.v2"
	"log"
	"os"
	"strconv"
	"strings"
)

// CurdCommand @Bean
type CurdCommand struct{}

type TableColumn struct {
	Name    string
	GoType  string
	Comment string
}

func (CurdCommand) Configure() command.Configure {
	return command.Configure{
		Name:        "make:curd",
		Description: "生成curd基础代码, 默认使用交互输入, 便捷调用 ",
		Input: command.Argument{
			Argument: []command.ArgParam{
				{
					Name:        "conn_name",
					Description: "连接名",
				},
				{
					Name:        "table_name",
					Description: "表名",
				},
			},
			Option: []command.ArgParam{
				{
					Name:        "config",
					Description: "配置文件",
					Default:     "@root/config/database.yaml",
				},
				{
					Name:        "go_out",
					Description: "生成文件到指定目录",
				},
				{
					Name:        "explain",
					Description: "说明",
				},
			},
		},
	}
}

func (CurdCommand) Execute(input command.Input) {
	root := getRootPath()
	file := input.GetOption("config")
	file = strings.Replace(file, "@root", root, 1)
	fileContext, _ := os.ReadFile(file)
	fileContext = SetEnv(fileContext)
	m := make(map[string]interface{})
	err := yaml.Unmarshal(fileContext, &m)
	if err != nil {
		log.Printf("配置解析错误:%v", err)
		return
	}
	connections := m["connections"].(map[interface{}]interface{})
	connName := input.GetArgument("conn_name")
	if _, ok := connections[connName]; !ok {
		log.Printf("没有找不到对应数据库连接")
		return
	}
	tableName := input.GetArgument("table_name")
	if tableName == "" {
		log.Printf("请输入表名")
		return
	}
	config := connections[connName].(map[interface{}]interface{})
	TableColumns := GetTableColumn(config, tableName)
	out := input.GetOption("go_out")
	if out == "" {
		log.Printf("请输入保存到目录地址")
		return
	}
	outUrl := root + "/app/http/admin/" + out + "/" + tableName
	_, err = os.Stat(outUrl)
	if os.IsNotExist(err) {
		err = os.MkdirAll(outUrl, 0766)
		if err != nil {
			log.Printf(err.Error())
			return
		}
	}
	protoUrl := root + "/protobuf/admin/" + out + "/" + tableName
	_, err = os.Stat(protoUrl)
	if os.IsNotExist(err) {
		err = os.MkdirAll(protoUrl, 0766)
		if err != nil {
			log.Printf(err.Error())
			return
		}
	}
	index := strings.Index(tableName, "_")
	contName := ""
	if index >= 0 {
		other := tableName[index+1:]
		contName = strings.ToUpper(tableName[:1]) + tableName[1:index] + strings.ToUpper(other[:1]) + other[1:]
	} else {
		contName = strings.ToUpper(tableName)
	}
	module := getModModule()
	//controller
	buildController(input, outUrl, module, contName)
	//del
	buildDel(input, outUrl, module, "del", contName)
	//get
	buildGet(input, outUrl, module, "get", contName)
	//post
	buildPost(input, outUrl, module, "post", contName)
	//put
	buildPut(input, outUrl, module, "put", contName)
	//proto
	buildProto(input, protoUrl, module, contName, TableColumns)
}

func buildController(input command.Input, outUrl string, module string, contName string) {
	cont := outUrl + "/" + input.GetOption("table_name") + "_controller.go"
	str := "package " + input.GetOption("table_name")
	pack := module + "/app/entity/" + input.GetOption("db_name")
	str += "\n\nimport (\n  \"" + pack + "\"\n)"
	str += "\n\n// " + input.GetOption("explain")
	str += "\ntype Controller struct {"
	str += "\n    orm *" + input.GetOption("db_name") + "." + contName + " `inject:\"\"`"
	str += "\n}"
	err := os.WriteFile(cont, []byte(str), 0766)
	if err != nil {
		log.Printf(err.Error())
		return
	}
}

func buildHead(table_name string, module string) string {
	str := "package " + table_name
	tm := []string{
		"github.com/gin-gonic/gin",
		module + "/app/common",
		module + "/app/providers",
		module + "/generate/proto/admin/" + table_name,
	}
	str += "\n\nimport ("
	for _, v := range tm {
		str += "\n	\"" + v + "\""
	}
	str += "\n)"
	return str
}

func buildDel(input command.Input, outUrl string, module string, name string, contName string) {
	cont := outUrl + "/del_action.go"
	str := buildHead(input.GetOption("table_name"), module)
	str += "\n\n// Del 删除数据 - " + input.GetOption("explain")
	str += "\nfunc (receiver *Controller) Del(req *" + input.GetOption("go_out") + "." + contName + "PutRequest, ctx *auth.Context) (*" +
		input.GetOption("go_out") + "." + contName + "PutRequest, error) {"
	str += "\n	id := common.GetParamId(ctx)"
	str += "\n	receiver.orm.Delete(id)"
	str += "\n 	return &" + input.GetOption("go_out") + "." + contName + "PutRequest{"
	str += "\n		Tip: \"OK\","
	str += "\n	}, nil"
	str += "\n}"
	str += handleValue(name, input.GetOption("go_out"), contName)
	err := os.WriteFile(cont, []byte(str), 0766)
	if err != nil {
		log.Printf(err.Error())
		return
	}
}

func buildGet(input command.Input, outUrl string, module string, name string, contName string) {
	cont := outUrl + "/get_action.go"
	str := buildHead(input.GetOption("table_name"), module)
	str += "\n\n// Get 列表查询 - " + input.GetOption("explain")
	str += fmt.Sprintf(`
func (receiver *Controller) Get(req *%v.%vGetRequest, ctx *auth.Context) (*%v.%vGetRequest, error) {
	list, total := receiver.orm.GetPaginate(req.Page, req.Limit)
	responseList := make([]*%v.%vInfo, 0)
	for _, cp := range list {
		responseList = append(responseList, &%v.%vInfo{})
	}
	return &%v.%vGetResponse{
		List:	responseList,
		Total:  uint32(total),
	}, nil
}
`,
		input.GetOption("go_out"),
		contName,
		input.GetOption("go_out"),
		contName,
		input.GetOption("table_name"),
		contName,
		input.GetOption("table_name"),
		contName,
		input.GetOption("go_out"),
		contName,
	)
	str += handleValue(name, input.GetOption("go_out"), contName)
	err := os.WriteFile(cont, []byte(str), 0766)
	if err != nil {
		log.Printf(err.Error())
		return
	}
}

func buildPost(input command.Input, outUrl string, module string, name string, contName string) {
	cont := outUrl + "/post_action.go"
	str := buildHead(input.GetOption("table_name"), module)
	str += "\n\n// Post 创建新数据 - " + input.GetOption("explain")
	str += "\nfunc (receiver *Controller) Post(req *" + input.GetOption("go_out") + "." + contName + "PostRequest, ctx *auth.Context) (*" +
		input.GetOption("go_out") + "." + contName + "PostRequest, error) {"
	str += "\n    id := int32(common.GetParamId(ctx))"
	str += "\n    has := receiver.orm.WhereId(id).First()"
	str += "\n    if has == nil {"
	str += "\n        return nil, nil"
	str += "\n    }"
	split := strings.Split(input.GetOption("table_name"), "_")
	dbFunc := ""
	for _, t := range split {
		dbFunc += strings.ToUpper(t[:1]) + t[1:]
	}
	str += "\n    data := " + input.GetOption("db_name") + "." + dbFunc + "{}"
	str += "\n    res := receiver.orm.Create(&data)"
	str += "\n    return &" + input.GetOption("go_out") + "." + contName + "PostResponse{}, res.Error"
	str += "\n}"
	str += handleValue(name, input.GetOption("go_out"), contName)
	err := os.WriteFile(cont, []byte(str), 0766)
	if err != nil {
		log.Printf(err.Error())
		return
	}
}

func buildPut(input command.Input, outUrl string, module string, name string, contName string) {
	cont := outUrl + "/put_action.go"
	str := buildHead(input.GetOption("table_name"), module)
	str += "\n\n// Put 更新数据 - " + input.GetOption("explain")
	str += "\nfunc (receiver *Controller) Put(req *" + input.GetOption("go_out") + "." + contName + "PostRequest, ctx *auth.Context) (*" +
		input.GetOption("go_out") + "." + contName + "PostRequest, error) {"
	str += "\n    id := int32(common.GetParamId(ctx))"
	str += "\n    has := receiver.orm.WhereId(id).First()"
	str += "\n    if has == nil {"
	str += "\n        return nil, nil"
	str += "\n    }"
	split := strings.Split(input.GetOption("table_name"), "_")
	dbFunc := ""
	for _, t := range split {
		dbFunc += strings.ToUpper(t[:1]) + t[1:]
	}
	str += "\n    receiver.orm.WhereId(id).Updates(&" + input.GetOption("db_name") + "." + dbFunc + "{})"
	str += "\n    return &" + input.GetOption("go_out") + "." + contName + "PutResponse{}, nil"
	str += "\n}"
	str += handleValue(name, input.GetOption("go_out"), contName)
	err := os.WriteFile(cont, []byte(str), 0766)
	if err != nil {
		log.Printf(err.Error())
		return
	}
}

func handleValue(name string, module string, contName string) string {
	name = strings.ToUpper(name[:1]) + name[1:]
	str := fmt.Sprintf(`
//GinHandle%v gin原始路由处理`, name)
	str += fmt.Sprintf(`
func (receiver *Controller) GinHandle%v(ctx *gin.Context) {
	req := &%v.%v%vRequest{}
	err := ctx.ShouleBind(req)
	if err != nil {
		providers.ErrorRequest(ctx, err)
		return
	}
	resp, err := receiver.%v(req, auth.NewContext(ctx))
	if err != nil {
		providers.ErrorResponse(ctx, err)
		return
	}
	providers.SuccessResponse(ctx, resp)
}
`,
		name,
		module,
		contName,
		name,
		name,
	)
	return str
}

func buildProto(input command.Input, protoUrl string, module string, contName string, column []TableColumn) {
	cont := protoUrl + "/" + input.GetOption("table_name") + ".proto"
	str := "// @Tag(\"form\");"
	str += "\nsyntax = \"proto3\";"
	str += "\n\npackage " + input.GetOption("table_name") + ";"
	str += "\n\nimport \"http_config.proto\";"
	str += "\n\noption go_package = \"" + module + "/generate/proto/admin/" + input.GetOption("go_out") + "\";"
	str += "\n// " + input.GetOption("explain") + "资源控制器"
	str += "\nservice " + contName + "{"
	str += "\n	// 需要登录"
	str += "\n  option (http.RouteGroup) = \"login\";"
	str += "\n  // " + input.GetOption("explain") + "列表"
	str += "\n  rpc Get(" + contName + "GetRequest) returns (" + contName + "GetResponse){"
	str += "\n  	option (http.Get) = \"/" + input.GetOption("go_out") + "/" + input.GetOption("table_name") + "\";"
	str += "\n  }"
	str += "\n  // " + input.GetOption("explain") + "创建"
	str += "\n  rpc Post(" + contName + "PostRequest) returns (" + contName + "PostResponse){"
	str += "\n  	option (http.Post) = \"/" + input.GetOption("go_out") + "/" + input.GetOption("table_name") + "\";"
	str += "\n  }"
	str += "\n  // " + input.GetOption("explain") + "更新"
	str += "\n  rpc Put(" + contName + "PutRequest) returns (" + contName + "PutResponse){"
	str += "\n  	option (http.Put) = \"/" + input.GetOption("go_out") + "/" + input.GetOption("table_name") + "/:id\";"
	str += "\n  }"
	str += "\n  // " + input.GetOption("explain") + "删除"
	str += "\n  rpc Del(" + contName + "PutRequest) returns (" + contName + "PutResponse){"
	str += "\n  	option (http.Get) = \"/" + input.GetOption("go_out") + "/" + input.GetOption("table_name") + "/:id\";"
	str += "\n  }"
	str += "\n}"
	str += "\nmessage " + contName + "GetRequest {"
	str += "\n  // 列表第几页，默认1"
	str += "\n  uint32 page = 1;"
	str += "\n  // 每页多少条数据"
	str += "\n  uint32 limit = 2;"
	str += "\n}"
	str += "\nmessage " + contName + "GetResponse {"
	str += "\n  // 数据列表"
	str += "\n  repeated " + contName + "Info list = 1;"
	str += "\n  // 符合条件总数量，计算多少页"
	str += "\n  uint32 total = 2;"
	str += "\n}"
	str += "\nmessage " + contName + "PostRequest {"
	str += "\n}"
	str += "\nmessage " + contName + "PostResponse {"
	str += "\n  // 提示语"
	str += "\n  string tip = 1;"
	str += "\n}"
	str += "\nmessage " + contName + "PutRequest {"
	str += "\n"
	str += "\n"
	str += "\n}"
	str += "\nmessage " + contName + "PutResponse {"
	str += "\n  // 提示语"
	str += "\n  string tip = 1;"
	str += "\n}"
	str += "\nmessage " + contName + "Info{"
	for i, v := range column {
		str += "\n    // " + v.Comment
		str += "\n    " + v.GoType + " " + v.Name + " = " + strconv.Itoa(i) + ";"
	}
	str += "\n}"
	err := os.WriteFile(cont, []byte(str), 0766)
	if err != nil {
		log.Printf(err.Error())
		return
	}
}

func GetTableColumn(config map[interface{}]interface{}, tableName string) []TableColumn {
	rows := &sql.Rows{}
	switch config["driver"] {
	case "mysql":
		rows, _ = orm.NewDb(config).GetDB().Query(`
SELECT COLUMN_NAME, DATA_TYPE, COLUMN_COMMENT
FROM information_schema.COLUMNS 
WHERE table_schema = DATABASE () AND table_name = $1
ORDER BY ORDINAL_POSITION ASC`, tableName)
	case "pgsql":
		rows, _ = pgorm.NewDb(config).GetDB().Query(`
SELECT i.column_name, i.udt_name, col_description(a.attrelid,a.attnum) as comment
FROM information_schema.columns as i 
LEFT JOIN pg_class as c on c.relname = i.table_name
LEFT JOIN pg_attribute as a on a.attrelid = c.oid and a.attname = i.column_name
WHERE table_schema = 'public' and i.table_name = $1;
`, tableName)
	default:
		panic("没有[" + config["driver"].(string) + "]的驱动")
	}
	defer rows.Close()
	var tableColumns []TableColumn
	for rows.Next() {
		var name, dataType, comment string
		var _comment *string
		_ = rows.Scan(
			&name,
			&dataType,
			&_comment,
		)
		if _comment == nil {
			comment = ""
		} else {
			comment = *_comment
		}
		switch config["driver"] {
		case "mysql":
			dataType = orm.TypeForMysqlToGo[dataType]
		case "pgsql":
			dataType = pgorm.PgTypeToGoType(dataType, name)
		}
		tableColumns = append(tableColumns, TableColumn{
			Name:    parser.StringToHump(name),
			GoType:  dataType,
			Comment: comment,
		})
	}
	return tableColumns
}
