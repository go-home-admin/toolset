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

	param.Module = getModule(input.GetOption("module"))
	param.Explain = getExplain(input.GetOption("explain"))
	outUrl := root + "/app/http/" + param.Module + "/" + param.TableName
	_, err = os.Stat(outUrl)
	if os.IsNotExist(err) {
		err = os.MkdirAll(outUrl, 0766)
		if err != nil {
			log.Printf(err.Error())
			return
		}
	}
	protoUrl := root + "/protobuf/" + param.Module + "/" + param.TableName
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

// 获取存放路径
func getModule(Module string) string {
	fmt.Printf("请输入存放路径: ")
	fmt.Scan(&Module)
	if Module == "" {
		return param.Module
	}
	return Module
}

func getExplain(Explain string) string {
	fmt.Printf("请输入说明: ")
	fmt.Scan(&Explain)
	return Explain
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
		module + "/generate/proto/" + param.Module,
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
	str += fmt.Sprintf(`
func (receiver *Controller) Del(req *%v.%vPutRequest, ctx *auth.Context) (*%v.%vPutRequest, error) {
	id := ctx.GetParamId()
	err := %v.NewOrm%v().Delete(id)
	return &%v.%vPutRequest{
		Tip: "OK",
	}, err.Error
}
`,
		param.Module,
		contName,
		param.Module,
		contName,
		param.CoonName,
		contName,
		param.Module,
		contName,
	)
	str += handleValue(name, param.Module, contName)
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
		param.Module,
		contName,
		param.Module,
		contName,
		param.CoonName,
		contName,
		param.Module,
		contName,
		param.Module,
		contName,
		co,
		param.Module,
		contName,
	)
	str += handleValue(name, param.Module, contName)
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
	split := strings.Split(param.TableName, "_")
	dbFunc := ""
	for _, t := range split {
		dbFunc += strings.ToUpper(t[:1]) + t[1:]
	}
	var pars string
	for _, v := range column {
		pars += fmt.Sprintf(`
			%v:		cp.%v,`,
			v.Name, v.Name,
		)
	}
	str += fmt.Sprintf(`
func (receiver *Controller) Post(req *%v.%vPostRequest, ctx *auth.Context) (*%v.%vPostResponse, error) {
	id := ctx.GetParamId()
	has := %v.NewOrm%v().WhereId(id).First()
	if has == nil {
		return nil, nil
	}
	data := %v.%v{%v
	}
	res := %v.NewOrm%v().Create(&data)
	return &%v.%vPostResponse{}, res.Error
}
`,
		param.Module,
		contName,
		param.Module,
		contName,
		param.CoonName,
		contName,
		param.CoonName,
		dbFunc,
		pars,
		param.CoonName,
		contName,
		param.Module,
		contName,
	)
	str += handleValue(name, param.Module, contName)
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
	split := strings.Split(param.TableName, "_")
	dbFunc := ""
	for _, t := range split {
		dbFunc += strings.ToUpper(t[:1]) + t[1:]
	}
	var pars string
	for _, v := range column {
		pars += fmt.Sprintf(`
			%v:		cp.%v,`,
			v.Name, v.Name,
		)
	}
	str += fmt.Sprintf(`
func (receiver *Controller) Put(req *%v.%vPostRequest, ctx *auth.Context) (*%v.%vPostRequest, error) {
	id := ctx.GetParamId()
	has := %v.NewOrm%v().WhereId(id).First()
	if has == nil {
		return nil, nil
	}
	err := %v.NewOrm%v().WhereId(id).Updates(&%v.%v{%v
	})
	return &%v.%vPutResponse{}, err.Error
}
`,
		param.Module,
		contName,
		param.Module,
		contName,
		param.CoonName,
		contName,
		param.CoonName,
		contName,
		param.CoonName,
		dbFunc,
		pars,
		param.Module,
		contName,
	)
	str += handleValue(name, param.Module, contName)
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
	var pars string
	for i, v := range column {
		pars += "\n	 // " + v.Comment
		if v.GoType == "database.Time" {
			v.GoType = "string"
		}
		pars += "\n  " + v.GoType + " " + v.Name + " = " + strconv.Itoa(i+1) + ";"
	}
	str += fmt.Sprintf(`
syntax = "proto3";

package %v;

import "http_config.proto";

option go_package = "%v/generate/proto/%v";
// %v资源控制器
service %v {
	// 需要登录
	option (http.RouteGroup) = "login";
	// %v列表
	rpc Get(%vGetRequest) returns (%vGetResponse){
		option (http.Get) = "/%v/%v";
	}
	// %v创建
	rpc Post(%vPostRequest) returns (%vPostResponse){
		option (http.Post) = "/%v/%v";
	}
	// %v更新
	rpc Put(%vPutRequest) returns (%vPutResponse){
		option (http.Put) = "/%v/%v/:id";
	}
	// %v删除
	rpc Del(%vDelRequest) returns (%vDelResponse){
		option (http.Get) = "/%v/%v/:id";
	}
}

message %vGetRequest {
	// 列表第几页，默认1
	uint32 page = 1;
	// 每页多少条数据
	uint32 limit = 2;
}

message %vGetResponse {
	// 数据列表
	repeated %vInfo list = 1;
	// 符合条件总数量，计算多少页
	uint32 total = 2;
}

message %vPostRequest {}

message %vPostResponse {
	// 提示语
	string tip = 1;
}

message %vPutRequest {}

message %vPutResponse {
	// 提示语
	string tip = 1;
}

message %vDelRequest {}

message %vDelResponse {
	// 提示语
	string tip = 1;
}

message %vInfo{
	%v
}
`,
		param.TableName,
		module,
		param.Module,
		param.Explain,
		contName,
		param.Explain,
		contName,
		contName,
		param.Module,
		param.TableName,
		param.Explain,
		contName,
		contName,
		param.Module,
		param.TableName,
		param.Explain,
		contName,
		contName,
		param.Module,
		param.TableName,
		param.Explain,
		contName,
		contName,
		param.Module,
		param.TableName,
		contName,
		contName,
		contName,
		contName,
		contName,
		contName,
		contName,
		contName,
		contName,
		contName,
		pars,
	)
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
