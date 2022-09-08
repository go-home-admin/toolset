package commands

import (
	_ "embed"
	"fmt"
	"github.com/ctfang/command"
	"github.com/go-home-admin/toolset/console/commands/orm"
	"github.com/go-home-admin/toolset/parser"
	"os"
	"path/filepath"
	"strings"
)

// MongoCommand @Bean
type MongoCommand struct{}

//go:embed mongo/orm.go.text
var ormTemplate string

func (MongoCommand) Configure() command.Configure {
	return command.Configure{
		Name:        "make:mongo",
		Description: "根据proto文件, 生成mongodb的orm源码",
		Input: command.Argument{
			Argument: []command.ArgParam{
				{
					Name:        "proto_file",
					Description: "一个以数据库名命名的proto文件",
				},
			},
			Option: []command.ArgParam{
				{
					Name:        "out",
					Description: "输出目录",
					Default:     "@root/app/entity",
				},
			},
		},
	}
}

func (MongoCommand) Execute(input command.Input) {
	root := getRootPath()
	file := input.GetArgument("proto_file")
	if file[0:1] != "/" && file[0:1] != "@" {
		file = "@root" + "/" + file
	}
	file = strings.Replace(file, "@root", root, 1)
	if !orm.IsExist(file) {
		panic("找不到proto文件")
	}
	out := input.GetOption("out")
	if out[0:1] != "/" && out[0:1] != "@" {
		out = "@root" + "/" + out
	}
	out = strings.Replace(out, "@root", root, 1)

	protoc, _ := parser.GetProtoFileParser(file)

	db := strings.Split(filepath.Base(file), ".")[0]
	out = out + "/" + db
	if !orm.IsExist(out) {
		_ = os.MkdirAll(out, 0766)
	}
	//生成枚举
	enumFile := out + "/" + "db_enum_gen.go"
	enumStr := "package " + db + "\n\n"
	for enumName, enumInfo := range protoc.Enums {
		enumStr += formatDoc(enumInfo.Doc, true) + "type " + enumName + " int32\n\n"
		enumStr += "const ("
		for _, attr := range enumInfo.Opt {
			enumStr += "\n" + formatDoc(attr.Doc, true)
			enumStr += enumName + "_" + attr.Name + " " + enumName + " = " + fmt.Sprint(attr.Num)
		}
		enumStr += "\n)\n\n"
	}
	err := os.WriteFile(enumFile, []byte(enumStr), 0766)
	if err != nil {
		panic("生成枚举文件出错：" + err.Error())
	}
	runOtherCommand("go", "fmt", enumFile)

	//生成结构文件
	structMap := make(map[string]int, 0)
	for structName, _ := range protoc.Messages {
		structMap[structName] = 1
	}
	structFile := out + "/" + "db_struct_gen.go"
	structStr := "package " + db + "\n"
	for structName, structInfo := range protoc.Messages {
		if strings.Index(structInfo.Doc, "@Struct") != -1 || strings.Index(structInfo.Doc, "@struct") != -1 {
			structStr += "\n" + formatDoc(structInfo.Doc, true)
			structStr += genOrmStruct(structName, structInfo.Attr, structMap) + "\n"
		}
	}
	err = os.WriteFile(structFile, []byte(structStr), 0766)
	if err != nil {
		panic("生成结构文件出错：" + err.Error())
	}
	runOtherCommand("go", "fmt", structFile)

	//生成ORM文件
	for tableName, tableInfo := range protoc.Messages {
		if strings.Index(tableInfo.Doc, "@Struct") != -1 || strings.Index(tableInfo.Doc, "@struct") != -1 {
			continue
		}
		str := ormTemplate
		tableNameSnake := parser.StringToSnake(tableName)
		ormFile := out + "/" + tableNameSnake + "_gen.go"
		for old, newStr := range map[string]string{
			"{database}":       db,
			"{tableName}":      tableName,
			"{tableNameSnake}": tableNameSnake,
			"{import}":         genImport(tableInfo.Attr),
			"{ormStruct}":      genOrmStruct(tableName, tableInfo.Attr, structMap),
			"{createdAt}":      genCreateAt(tableInfo.Attr),
			"{updatedAt}":      genUpdatedAt(tableInfo.Attr),
			"{where}":          genWhere(tableName, tableInfo.Attr),
		} {
			str = strings.ReplaceAll(str, old, newStr)
		}
		err = os.WriteFile(ormFile, []byte(str), 0766)
		if err != nil {
			panic("生成ORM文件出错：" + err.Error())
		}
		runOtherCommand("go", "fmt", ormFile)
	}
}

func genOrmStruct(tableName string, columns []parser.Attr, structMap map[string]int) string {
	str := "type " + tableName + " struct {"
	str += "\n\tId string `bson:\"_id\" form:\"id\" json:\"id,omitempty\"`"
	for _, column := range columns {
		if column.Name == "id" || column.Name == "_id" {
			continue
		}
		fieldName := parser.StringToHump(column.Name)
		str += fmt.Sprintf("\n\t%v %v `%v` %v", fieldName, getGoType(column, structMap), genGormTag(column), formatDoc(column.Doc))
	}
	return str + "\n}"
}

func genGormTag(column parser.Attr) string {
	return strings.ReplaceAll(`bson:"{field}" form:"{field}" json:"{field},omitempty"`, "{field}", column.Name)
}

func formatDoc(doc string, wrap ...bool) string {
	if doc == "" {
		return ""
	}
	end := ""
	if len(wrap) > 0 && wrap[0] == true {
		end = "\n"
	}
	return "// " + strings.ReplaceAll(strings.ReplaceAll(doc, "@Struct", ""), "@struct", "") + end
}

func getGoType(column parser.Attr, structMap map[string]int) string {
	t := column.Ty
	switch t {
	case "double", "float":
		t = "float64"
	case "oneof":
		t = "interface{}"
	}
	if _, ok := structMap[t]; ok {
		t = "*" + t
	}
	if column.Repeated {
		t = "[]" + t
	}
	return t
}

func genImport(columns []parser.Attr) string {
	str := `"context"
	"errors"
	provider "github.com/go-home-admin/home/bootstrap/providers/mongo"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/mgo.v2/bson"
	"strings"`
	for _, column := range columns {
		if column.Name == "created_at" || column.Name == "updated_at" {
			str += "\n\"time\"\n"
		}
	}
	return str
}

func genCreateAt(columns []parser.Attr) string {
	str := ""
	for _, column := range columns {
		if column.Name == "created_at" || column.Name == "updated_at" {
			columnName := parser.StringToHump(column.Name)
			if strings.Contains(column.Ty, "int") {
				timeStr := "time.Now().Unix()"
				if column.Ty != "int64" {
					timeStr = column.Ty + "(" + timeStr + ")"
				}
				str += fmt.Sprintf("if data.%v == 0 {\ndata.%v = %v\n}\n", columnName, columnName, timeStr)
			} else if column.Ty == "string" {
				str += strings.ReplaceAll("if data.{field} == \"\" {\ndata.{field} = time.Now().Format(\"2006-01-02 15:04:05\")\n}\n", "{field}", columnName)
			}
		}
	}
	return str
}

func genUpdatedAt(columns []parser.Attr) string {
	str := ""
	for _, column := range columns {
		if column.Name == "updated_at" {
			if strings.Contains(column.Ty, "int") {
				str += "if data[\"UpdatedAt\"] == 0 {\ndata[\"UpdatedAt\"] = time.Now().Unix()\n}\n"
			} else if column.Ty == "string" {
				str += "if _,ok := data[\"UpdatedAt\"]; !ok {\ndata[\"UpdatedAt\"] = time.Now().Format(\"2006-01-02 15:04:05\")\n}\n"
			}
		}
	}
	return str
}

func genWhere(tableName string, columns []parser.Attr) string {
	var supportType = map[string]int{
		"int":     1,
		"int32":   1,
		"uint32":  1,
		"int64":   1,
		"float32": 1,
		"float64": 1,
		"string":  1,
		"bool":    1,
	}
	template := "func (receiver *Orm%v) Where%v(value %v) *Orm%v {\nreturn receiver.Where(\"%v\", value)\n}\n\n"
	str := ""
	for _, column := range columns {
		if _, ok := supportType[column.Ty]; ok {
			field := parser.StringToHump(column.Name)
			str += fmt.Sprintf(template, tableName, field, column.Ty, tableName, column.Name)
		}
	}
	return str
}
