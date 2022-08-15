package commands

import (
	"database/sql"
	"fmt"
	"github.com/ctfang/command"
	"github.com/go-home-admin/toolset/console/commands/orm"
	"github.com/go-home-admin/toolset/console/commands/pgorm"
	"github.com/go-home-admin/toolset/parser"
	"github.com/joho/godotenv"
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

type Param struct {
	CoonName  string
	TableName string
	Module    string
	Explain   string
	DbName    string
}

var param Param

func (CurdCommand) Configure() command.Configure {
	return command.Configure{
		Name:        "make:curd",
		Description: "生成curd基础代码, 默认使用交互输入, 便捷调用 ",
		Input: command.Argument{
			Option: []command.ArgParam{
				{
					Name:        "conn_name",
					Description: "连接名",
				},
				{
					Name:        "table_name",
					Description: "表名",
				},
				{
					Name:        "config",
					Description: "配置文件",
					Default:     "@root/config/database.yaml",
				},
				{
					Name:        "module",
					Description: "模块名称, 默认: admin",
					Default:     "admin",
				},
				{
					Name:        "explain",
					Description: "生成的注释, 默认为表注释",
					Call: func(val string, c *command.Console) (string, bool) {
						return "", true
					},
				},
			},
		},
	}
}

func (CurdCommand) Execute(input command.Input) {
	root := getRootPath()
	err := godotenv.Load(root + "/.env")
	if err != nil {
		fmt.Println(root + "/.env" + "文件不存在, 无法加载环境变量")
	}
	file := input.GetOption("config")
	file = strings.Replace(file, "@root", root, 1)
	fileContext, _ := os.ReadFile(file)
	fileContext = SetEnv(fileContext)
	m := make(map[string]interface{})
	err = yaml.Unmarshal(fileContext, &m)
	if err != nil {
		log.Printf("配置解析错误:%v", err)
		return
	}
	connections := m["connections"].(map[interface{}]interface{})
	param.CoonName = getConnName(input.GetOption("conn_name"), connections)
	param.TableName = getTableName(input.GetOption("table_name"), connections[param.CoonName])

	config := connections[param.CoonName].(map[interface{}]interface{})
	TableColumns := GetTableColumn(config, param.TableName)

	module := input.GetOption("module")
	outUrl := root + "/app/http/" + module + "/" + param.TableName
	_, err = os.Stat(outUrl)
	if os.IsNotExist(err) {
		err = os.MkdirAll(outUrl, 0766)
		if err != nil {
			log.Printf(err.Error())
			return
		}
	}
	protoUrl := root + "/protobuf/" + module + "/" + param.TableName
	_, err = os.Stat(protoUrl)
	if os.IsNotExist(err) {
		err = os.MkdirAll(protoUrl, 0766)
		if err != nil {
			log.Printf(err.Error())
			return
		}
	}
	index := strings.Index(param.TableName, "_")
	contName := ""
	if index >= 0 {
		other := param.TableName[index+1:]
		contName = strings.ToUpper(param.TableName[:1]) + param.TableName[1:index] + strings.ToUpper(other[:1]) + other[1:]
	} else {
		contName = strings.ToUpper(param.TableName)
	}
	goMod := getModModule()
	//controller
	buildController(param, outUrl, goMod, contName)
	//del
	buildDel(param, outUrl, goMod, "del", contName)
	//get
	buildGet(param, outUrl, goMod, "get", contName, TableColumns)
	//post
	buildPost(param, outUrl, goMod, "post", contName, TableColumns)
	//put
	buildPut(param, outUrl, goMod, "put", contName, TableColumns)
	//proto
	buildProto(param, protoUrl, goMod, contName, TableColumns)
}

// 获取连接名称
func getConnName(connName string, connections map[interface{}]interface{}) string {
	if connName == "" {
		var got int
		gotName := make(map[int]string)
		fmt.Printf("请选中以下连接数据库配置\n")
		for name, m := range connections {
			conf := m.(map[interface{}]interface{})
			driver := conf["driver"]
			if driver == "mysql" || driver == "pgsql" {
				got++
				gotName[got] = name.(string)
				fmt.Printf("%v: %v\n", got, name)
			}
		}
		if len(gotName) == 1 {
			got = 1
			fmt.Printf("只有一个数据库, 已经自动选中: 1\n")
		} else {
			fmt.Printf("请输入数字: ")
			fmt.Scan(&got)
		}
		connName = gotName[got]
	}

	if _, ok := connections[connName]; !ok {
		panic("没有找不到对应数据库连接")
	}

	return connName
}

func getTableName(tableName string, m interface{}) string {
	conf := m.(map[interface{}]interface{})
	if tableName == "" {
		fmt.Printf("未指定表, 可以使用以下的表生成\n")
		tables := make(map[int]string)
		switch conf["driver"] {
		case "mysql":
			db := orm.NewDb(conf)
			rows, _ := db.GetDB().Query("SELECT A.TABLE_NAME as name FROM INFORMATION_SCHEMA.COLUMNS A WHERE A.TABLE_SCHEMA = ? GROUP BY TABLE_NAME", conf["database"].(string))
			defer rows.Close()
			i := 0
			for rows.Next() {
				i++
				var name string
				rows.Scan(&name)
				tables[i] = name
				fmt.Printf("%v: %v\n", i, name)
			}
		case "pgsql":

		}

		fmt.Printf("请输入数字: ")
		var got int
		fmt.Scan(&got)
		tableName = tables[got]
	}

	return tableName
}

func buildController(param Param, outUrl string, module string, contName string) {
	cont := outUrl + "/" + param.TableName + "_controller.go"
	str := "package " + param.TableName
	str += "\n\n// " + param.Explain
	str += "\ntype Controller struct {}"
	err := os.WriteFile(cont, []byte(str), 0766)
	if err != nil {
		log.Printf(err.Error())
		return
	}
}

func buildHead(tableName string, module string, name string) string {
	str := "package " + tableName
	tm := []string{
		"github.com/gin-gonic/gin",
		module + "/app/common/auth",
		module + "/app/entity/" + param.CoonName,
		module + "/app/providers",
		module + "/generate/proto/admin",
		module + "/home/app/http",
	}
	str += "\n\nimport ("
	for _, v := range tm {
		str += "\n	\"" + v + "\""
	}
	str += "\n)"
	return str
}

func buildDel(param Param, outUrl string, module string, name string, contName string) {
	cont := outUrl + "/del_action.go"
	str := buildHead(param.TableName, module, name)
	str += "\n\n// Del 删除数据 - " + param.Explain
	str += "\nfunc (receiver *Controller) Del(req *admin" + "." + contName + "PutRequest, ctx *auth.Context) (*admin" + "." + contName + "PutRequest, error) {"
	str += "\n	id := ctx.GetId()"
	str += "\n	err := " + param.CoonName + ".NewOrm" + contName + ".Delete(id)"
	str += "\n 	return &admin" + "." + contName + "PutRequest{"
	str += "\n		Tip: \"OK\","
	str += "\n	}, err.Error"
	str += "\n}"
	str += "\n"
	str += handleValue(name, "admin", contName)
	err := os.WriteFile(cont, []byte(str), 0766)
	if err != nil {
		log.Printf(err.Error())
		return
	}
}

func buildGet(param Param, outUrl string, module string, name string, contName string, column []TableColumn) {
	cont := outUrl + "/get_action.go"
	str := buildHead(param.TableName, module, name)
	str += "\n\n// Get 列表查询 - " + param.Explain
	co := ""
	for _, v := range column {
		co += fmt.Sprintf(`
			%v:		cp.%v,`,
			v.Name, v.Name)
	}
	str += fmt.Sprintf(`
func (receiver *Controller) Get(req *%v.%vGetRequest, ctx *auth.Context) (*%v.%vGetResponse, error) {
	list, total := %v.NewOrm%v().Paginate(int(req.Page), int(req.Limit))
	responseList := make([]*%v.%vInfo, 0)
	for _, cp := range list {
		responseList = append(responseList, &%v.%vInfo{%v
		})
	}
	return &%v.%vGetResponse{
		List:	responseList,
		Total:  uint32(total),
	}, nil
}
`,
		"admin",
		contName,
		"admin",
		contName,
		param.CoonName,
		contName,
		"admin",
		contName,
		"admin",
		contName,
		co,
		"admin",
		contName,
	)
	str += handleValue(name, "admin", contName)
	err := os.WriteFile(cont, []byte(str), 0766)
	if err != nil {
		log.Printf(err.Error())
		return
	}
}

func buildPost(param Param, outUrl string, module string, name string, contName string, column []TableColumn) {
	cont := outUrl + "/post_action.go"
	str := buildHead(param.TableName, module, name)
	str += "\n\n// Post 创建新数据 - " + param.Explain
	str += "\nfunc (receiver *Controller) Post(req *admin" + "." + contName + "PostRequest, ctx *auth.Context) (*admin" + "." + contName + "PostResponse, error) {"
	str += "\n    id := int32(common.GetParamId(ctx))"
	str += "\n    has := receiver.orm.WhereId(id).First()"
	str += "\n    if has == nil {"
	str += "\n        return nil, nil"
	str += "\n    }"
	split := strings.Split(param.TableName, "_")
	dbFunc := ""
	for _, t := range split {
		dbFunc += strings.ToUpper(t[:1]) + t[1:]
	}
	str += "\n    data := " + param.CoonName + "." + dbFunc + "{"
	for _, v := range column {
		str += "\n    	" + v.Name + ":		" + "cp." + v.Name + ","
	}
	str += "\n }"
	str += "\n    res := receiver.orm.Create(&data)"
	str += "\n    return &admin" + "." + contName + "PostResponse{}, res.Error"
	str += "\n}"
	str += handleValue(name, "admin", contName)
	err := os.WriteFile(cont, []byte(str), 0766)
	if err != nil {
		log.Printf(err.Error())
		return
	}
}

func buildPut(param Param, outUrl string, module string, name string, contName string, column []TableColumn) {
	cont := outUrl + "/put_action.go"
	str := buildHead(param.TableName, module, name)
	str += "\n\n// Put 更新数据 - " + param.Explain
	str += "\nfunc (receiver *Controller) Put(req *admin" + "." + contName + "PostRequest, ctx *auth.Context) (*admin" + "." + contName + "PostRequest, error) {"
	str += "\n    id := int32(common.GetParamId(ctx))"
	str += "\n    has := receiver.orm.WhereId(id).First()"
	str += "\n    if has == nil {"
	str += "\n        return nil, nil"
	str += "\n    }"
	split := strings.Split(param.TableName, "_")
	dbFunc := ""
	for _, t := range split {
		dbFunc += strings.ToUpper(t[:1]) + t[1:]
	}
	str += "\n    err := receiver.orm.WhereId(id).Updates(&" + param.CoonName + "." + dbFunc + "{"
	for _, v := range column {
		str += "\n    " + v.Name + ":		" + "cp." + v.Name + ","
	}
	str += "})"
	str += "\n    return &admin" + "." + contName + "PutResponse{}, err.Error"
	str += "\n}"
	str += handleValue(name, "admin", contName)
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

func buildProto(param Param, protoUrl string, module string, contName string, column []TableColumn) {
	cont := protoUrl + "/" + param.TableName + ".proto"
	str := "// @Tag(\"form\");"
	str += "\nsyntax = \"proto3\";"
	str += "\n\npackage " + param.TableName + ";"
	str += "\n\nimport \"http_config.proto\";"
	str += "\n\noption go_package = \"" + module + "/generate/proto/admin\"" + ";"
	str += "\n// " + param.Explain + "资源控制器"
	str += "\nservice " + contName + "{"
	str += "\n	// 需要登录"
	str += "\n  option (http.RouteGroup) = \"login\";"
	str += "\n  // " + param.Explain + "列表"
	str += "\n  rpc Get(" + contName + "GetRequest) returns (" + contName + "GetResponse){"
	str += "\n  	option (http.Get) = \"/admin" + "/" + param.TableName + "\";"
	str += "\n  }"
	str += "\n  // " + param.Explain + "创建"
	str += "\n  rpc Post(" + contName + "PostRequest) returns (" + contName + "PostResponse){"
	str += "\n  	option (http.Post) = \"/admin" + "/" + param.TableName + "\";"
	str += "\n  }"
	str += "\n  // " + param.Explain + "更新"
	str += "\n  rpc Put(" + contName + "PutRequest) returns (" + contName + "PutResponse){"
	str += "\n  	option (http.Put) = \"/admin" + "/" + param.TableName + "/:id\";"
	str += "\n  }"
	str += "\n  // " + param.Explain + "删除"
	str += "\n  rpc Del(" + contName + "PutRequest) returns (" + contName + "PutResponse){"
	str += "\n  	option (http.Get) = \"/admin" + "/" + param.TableName + "/:id\";"
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
		if v.GoType == "database.Time" {
			v.GoType = "string"
		}
		str += "\n    " + v.GoType + " " + v.Name + " = " + strconv.Itoa(i+1) + ";"
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
	var err error
	switch config["driver"] {
	case "mysql":
		rows, err = orm.NewDb(config).GetDB().Query(`
SELECT COLUMN_NAME, DATA_TYPE, COLUMN_COMMENT
FROM information_schema.COLUMNS 
WHERE table_schema = DATABASE () AND table_name = ?
ORDER BY ORDINAL_POSITION ASC`, tableName)
	case "pgsql":
		db := pgorm.NewDb(config)
		rows, err = db.GetDB().Query(`
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
	if err != nil {
		panic("数据库连接失败或没有找到对应的表")
	}
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
