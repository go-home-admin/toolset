package orm

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/go-home-admin/home/bootstrap/services"
	"github.com/go-home-admin/toolset/parser"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// IsExist checks whether a file or directory exists.
// It returns false when the file or directory does not exist.
func IsExist(f string) bool {
	_, err := os.Stat(f)
	return err == nil || os.IsExist(err)
}

type Conf map[interface{}]interface{}

type Relationship struct {
	Type              string `json:"type"`               //关联类型：belongs_to、has_one、has_many、many2many
	Table             string `json:"table"`              //关联表名
	Alias             string `json:"alias"`              //别名（可不声明，默认用表名）
	ForeignKey        string `json:"foreign_key"`        //外键（可不声明，默认为'id'或'表名_id'）
	ReferenceKey      string `json:"reference_key"`      //引用键（可不声明，默认为'id'或'表名_id'）
	RelationshipTable string `json:"relationship_table"` //当many2many时，连接表名
	JoinForeignKey    string `json:"join_foreign_key"`   //当many2many时，本表在连接表的外键
	JoinTargetKey     string `json:"join_target_key"`    //当many2many时，关联表在连接表的外键
}

func GenMysql(name string, conf Conf, out string) {
	if !IsExist(out) {
		os.MkdirAll(out, 0766)
	}

	db := NewDb(conf)
	tableInfos := db.tableColumns()
	tableColumns := tableInfos.Columns

	root, _ := os.Getwd()
	file, err := os.ReadFile(root + "/config/database/" + name + ".json")
	var relationship map[string][]*Relationship
	if err == nil {
		err = json.Unmarshal(file, &relationship)
		if err != nil {
			panic("表关系JSON文件配置出错：" + err.Error())
		}
	}

	// 计算import
	imports := getImports(tableInfos.Infos, tableColumns)
	for table, columns := range tableColumns {
		tableConfig := tableInfos.Infos[table]
		mysqlTableName := parser.StringToSnake(table)
		file := out + "/" + mysqlTableName

		if _, err := os.Stat(file + "_lock.go"); !os.IsNotExist(err) {
			continue
		}

		str := "package " + name
		str += "\nimport (" + imports[table] + "\n)"
		str += "\n" + genOrmStruct(table, columns, conf, relationship[table])

		var baseFunStr string
		if tableConfig.IsSub() {
			baseFunStr = baseMysqlSubinfoStr
		} else {
			baseFunStr = baseMysqlFuncStr
		}
		for old, newStr := range map[string]string{
			"MysqlTableName": parser.StringToHump(table),
			"{table_name}":   table,
			"{db}":           name,
		} {
			baseFunStr = strings.ReplaceAll(baseFunStr, old, newStr)
		}

		str += baseFunStr
		str += genFieldFunc(table, columns)
		str += genListFunc(table, columns)
		err := os.WriteFile(file+"_gen.go", []byte(str), 0766)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func genListFunc(table string, columns []tableColumn) string {
	TableName := parser.StringToHump(table)
	str := "\ntype " + TableName + "List []*" + TableName
	for _, column := range columns {
		// 索引，或者枚举字段
		if strInStr(column.COLUMN_NAME, []string{"id", "code"}) || strInStr(column.COLUMN_COMMENT, []string{"@index"}) {
			goType := column.GoType
			if *column.IS_NULLABLE == "YES" {
				goType = "*" + goType
			}
			str += "\nfunc (l " + TableName + "List) Get" + column.ColumnName + "List() []" + goType + " {" +
				"\n\tgot := make([]" + goType + ", 0)\n\tfor _, val := range l {" +
				"\n\t\tgot = append(got, val." + column.ColumnName + ")" +
				"\n\t}" +
				"\n\treturn got" +
				"\n}"

			str += "\nfunc (l " + TableName + "List) Get" + column.ColumnName + "Map() map[" + goType + "]*" + TableName + " {" +
				"\n\tgot := make(map[" + goType + "]*" + TableName + ")\n\tfor _, val := range l {" +
				"\n\t\tgot[val." + column.ColumnName + "] = val" +
				"\n\t}" +
				"\n\treturn got" +
				"\n}"
		}
	}
	return str
}

func genFieldFunc(table string, columns []tableColumn) string {
	TableName := parser.StringToHump(table)

	str := ""
	for _, column := range columns {
		// 等于函数
		str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "(val " + column.GoType + ") *Orm" + TableName + " {" +
			"\n\torm.db.Where(\"`" + column.COLUMN_NAME + "` = ?\", val)" +
			"\n\treturn orm" +
			"\n}"

		if column.COLUMN_KEY != "" {
			goType := column.GoType
			if *column.IS_NULLABLE == "YES" {
				goType = "*" + goType
			}
			// if 主键, 生成In, > <
			if column.COLUMN_KEY == "PRI" {
				str += "\nfunc (orm *Orm" + TableName + ") InsertGet" + column.ColumnName + "(row *" + TableName + ") " + goType + " {" +
					"\n\torm.db.Create(row)" +
					"\n\treturn row." + column.ColumnName +
					"\n}"
			}

			str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "In(val []" + column.GoType + ") *Orm" + TableName + " {" +
				"\n\torm.db.Where(\"`" + column.COLUMN_NAME + "` IN ?\", val)" +
				"\n\treturn orm" +
				"\n}"

			str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "Gt(val " + column.GoType + ") *Orm" + TableName + " {" +
				"\n\torm.db.Where(\"`" + column.COLUMN_NAME + "` > ?\", val)" +
				"\n\treturn orm" +
				"\n}"
			str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "Gte(val " + column.GoType + ") *Orm" + TableName + " {" +
				"\n\torm.db.Where(\"`" + column.COLUMN_NAME + "` >= ?\", val)" +
				"\n\treturn orm" +
				"\n}"

			str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "Lt(val " + column.GoType + ") *Orm" + TableName + " {" +
				"\n\torm.db.Where(\"`" + column.COLUMN_NAME + "` < ?\", val)" +
				"\n\treturn orm" +
				"\n}"
			str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "Lte(val " + column.GoType + ") *Orm" + TableName + " {" +
				"\n\torm.db.Where(\"`" + column.COLUMN_NAME + "` <= ?\", val)" +
				"\n\treturn orm" +
				"\n}"
		} else {
			// 索引，或者枚举字段
			if strInStr(column.COLUMN_NAME, []string{"id", "code", "status", "state"}) {
				// else if 名称存在 id, code, status 生成in操作
				str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "In(val []" + column.GoType + ") *Orm" + TableName + " {" +
					"\n\torm.db.Where(\"`" + column.COLUMN_NAME + "` IN ?\", val)" +
					"\n\treturn orm" +
					"\n}"

				str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "Ne(val " + column.GoType + ") *Orm" + TableName + " {" +
					"\n\torm.db.Where(\"`" + column.COLUMN_NAME + "` <> ?\", val)" +
					"\n\treturn orm" +
					"\n}"
			}
			// 时间字段
			if strInStr(column.COLUMN_NAME, []string{"created", "updated", "time", "_at"}) || (column.GoType == "database.Time") {
				str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "Between(begin " + column.GoType + ", end " + column.GoType + ") *Orm" + TableName + " {" +
					"\n\torm.db.Where(\"`" + column.COLUMN_NAME + "` BETWEEN ? AND ?\", begin, end)" +
					"\n\treturn orm" +
					"\n}"

				str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "Lte(val " + column.GoType + ") *Orm" + TableName + " {" +
					"\n\torm.db.Where(\"`" + column.COLUMN_NAME + "` <= ?\", val)" +
					"\n\treturn orm" +
					"\n}"

				str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "Gte(val " + column.GoType + ") *Orm" + TableName + " {" +
					"\n\torm.db.Where(\"`" + column.COLUMN_NAME + "` >= ?\", val)" +
					"\n\treturn orm" +
					"\n}"
			}
		}
	}

	return str
}

func strInStr(s string, in []string) bool {
	for _, sub := range in {
		if strings.Index(s, sub) != -1 {
			return true
		}
	}
	return false
}

//go:embed mysql.go.subtext
var baseMysqlSubinfoStr string

//go:embed mysql.go.text
var baseMysqlFuncStr string

// 字段类型引入
var alias = map[string]string{
	"database": "github.com/go-home-admin/home/bootstrap/services/database",
}

// 获得 table => map{alias => github.com/*}
func getImports(infos map[string]TableInfos, tableColumns map[string][]tableColumn) map[string]string {
	got := make(map[string]string)
	for table, columns := range tableColumns {
		// 初始引入
		tm := map[string]string{
			"strings":      "",
			"gorm.io/gorm": "gorm",
			"github.com/go-home-admin/home/bootstrap/providers": "providers",
			"github.com/sirupsen/logrus":                        "logrus",
			"database/sql":                                      "sql",
			"github.com/go-home-admin/home/app":                 "home",
		}
		if infos[table].IsSub() {
			delete(tm, "github.com/go-home-admin/home/bootstrap/providers")
		}

		for _, column := range columns {
			index := strings.Index(column.GoType, ".")
			if index != -1 {
				as := column.GoType[:index]
				importStr := alias[as]
				tm[importStr] = as
			}
		}
		got[table] = parser.GetImportStrForMap(tm)
	}

	return got
}

func genOrmStruct(table string, columns []tableColumn, conf Conf, relationships []*Relationship) string {
	TableName := parser.StringToHump(table)
	config := services.NewConfig(conf)
	deletedField := config.GetString("deleted_field")
	hasField := make(map[string]bool)
	str := `type {TableName} struct {`
	for _, column := range columns {
		p := ""
		if *column.IS_NULLABLE == "YES" && !(column.COLUMN_NAME == "deleted_at" && column.GoType == "database.Time") {
			p = "*"
		}
		if column.GoType == "database.Time" && (column.COLUMN_NAME == deletedField || (deletedField == "" && column.COLUMN_NAME == "deleted_at")) {
			column.GoType = "gorm.DeletedAt"
		}

		// 使用注释@Type(int), 强制设置生成的go struct 属性 类型
		if i := strings.Index(column.COLUMN_COMMENT, "@type("); i != -1 {
			s := column.COLUMN_COMMENT[i+6:]
			e := strings.Index(s, ")")
			column.GoType = s[:e]
		}

		hasField[column.COLUMN_NAME] = true
		fieldName := parser.StringToHump(column.COLUMN_NAME)
		str += fmt.Sprintf("\n\t%v %v%v`%v json:\"%v\"` // %v", fieldName, p, column.GoType, genGormTag(column, conf), column.COLUMN_NAME, strings.ReplaceAll(column.COLUMN_COMMENT, "\n", " "))
	}
	// 表关系
	if len(relationships) > 0 {
		for _, relationship := range relationships {
			switch relationship.Type {
			case "belongs_to", "has_one", "has_many", "many2many":
			default:
				panic("with: belongs_to,has_one,has_many,many2many")
			}
			tbName := "*" + parser.StringToHump(relationship.Table)
			if relationship.Type == "has_many" || relationship.Type == "many2many" {
				tbName = "[]" + tbName
			}
			fieldName := parser.StringToHump(relationship.Table)
			if relationship.Alias != "" {
				fieldName = parser.StringToHump(relationship.Alias)
			}
			str += fmt.Sprintf("\n\t%v %v", fieldName, tbName)
			if relationship.ForeignKey != "" || relationship.ReferenceKey != "" || relationship.Type == "many2many" {
				str += " `gorm:\""
				if relationship.Type == "many2many" {
					if relationship.RelationshipTable == "" {
						panic("表" + relationship.Table + "的many2many必须声明连接表")
					}
					str += "many2many:" + relationship.RelationshipTable + ";"
					if relationship.JoinForeignKey != "" {
						str += "joinForeignKey:" + relationship.JoinForeignKey + ";"
					}
					if relationship.JoinTargetKey != "" {
						str += "joinReferences:" + relationship.JoinTargetKey + ";"
					}
				}
				if relationship.ForeignKey != "" {
					str += "foreignKey:" + relationship.ForeignKey + ";"
				}
				if relationship.ReferenceKey != "" {
					str += "references:" + relationship.ReferenceKey + ";"
				}
				str += "\"`"
			}
		}
	}
	str += "\n}\n\n"
	// 声明表字段
	str += "var (\n"
	for _, column := range columns {
		str += fmt.Sprintf("{TableName}Field%s = \"%s\"\n", parser.StringToHump(column.COLUMN_NAME), column.COLUMN_NAME)
	}
	str += ")"

	str = strings.ReplaceAll(str, "{TableName}", TableName)
	return "\n" + str + "\n"
}

func genGormTag(column tableColumn, conf Conf) string {
	var arr []string
	// 字段
	arr = append(arr, "column:"+column.COLUMN_NAME)
	switch column.EXTRA {
	case "":
	case "auto_increment":
		arr = append(arr, "autoIncrement")
	case "on update current_timestamp()":
		arr = append(arr, "autoUpdateTime")
	}

	// 类型ing
	arr = append(arr, "type:"+column.mysql.COLUMN_TYPE)
	// 索引
	if column.Index != nil {
		for _, index := range column.Index {
			switch index.INDEX_NAME {
			case "PRIMARY":
				arr = append(arr, "primaryKey")
			default:
				iStr := fmt.Sprintf("index:%v,class:%v", index.INDEX_NAME, index.INDEX_TYPE)
				if index.NON_UNIQUE == "0" {
					iStr += ",unique"
				}
				if index.COMMENT != "" {
					iStr += ",comment:" + index.COMMENT
				}
				arr = append(arr, iStr)
			}
		}
	}
	// default
	if column.COLUMN_DEFAULT != nil {
		arr = append(arr, "default:"+*column.COLUMN_DEFAULT)
	}
	// created_at & updated_at
	if field, ok := conf["created_field"]; ok && field == column.ColumnName {
		arr = append(arr, "autoCreateTime")
	}
	if field, ok := conf["updated_field"]; ok && field == column.ColumnName {
		arr = append(arr, "autoUpdateTime")
	}

	if column.COLUMN_COMMENT != "" {
		arr = append(arr, fmt.Sprintf("comment:'%v'", strings.ReplaceAll(column.COLUMN_COMMENT, "'", "")))
	}
	str := ""
	for i := 0; i < len(arr)-1; i++ {
		str += arr[i] + ";"
	}
	str += "" + arr[len(arr)-1]
	return "gorm:\"" + str + "\""
}

type DB struct {
	db *sql.DB
}

func (d *DB) GetDB() *sql.DB {
	return d.db
}

// 获取所有表信息
// 过滤分表信息, table_{1-9} 只返回table
func (d *DB) tableColumns() TableInfo {
	var sqlStr = `SELECT
	TABLE_CATALOG,
	TABLE_SCHEMA,
	TABLE_NAME,
	COLUMN_NAME,
	ORDINAL_POSITION,
	COLUMN_DEFAULT,
	IS_NULLABLE,
	DATA_TYPE,
	CHARACTER_MAXIMUM_LENGTH,
	CHARACTER_OCTET_LENGTH,
	NUMERIC_PRECISION,
	NUMERIC_SCALE,
	DATETIME_PRECISION,
	CHARACTER_SET_NAME,
	COLLATION_NAME,
	COLUMN_TYPE,
	COLUMN_KEY,
	EXTRA,
	PRIVILEGES,
	COLUMN_COMMENT,
	GENERATION_EXPRESSION
FROM
	information_schema.COLUMNS 
WHERE
	table_schema = DATABASE () 
ORDER BY
	TABLE_NAME ASC,
	ORDINAL_POSITION ASC`

	rows, err := d.db.Query(sqlStr)
	if err != nil {
		log.Println("Error reading table information: ", err.Error())
		return TableInfo{}
	}

	defer rows.Close()
	tableIndex := d.tableIndex()
	tableColumns := make(map[string][]tableColumn)
	for rows.Next() {
		col := tableColumn{}
		err = rows.Scan(
			&col.TABLE_CATALOG,
			&col.TABLE_SCHEMA,
			&col.TABLE_NAME,
			&col.COLUMN_NAME,
			&col.ORDINAL_POSITION,
			&col.COLUMN_DEFAULT,
			&col.IS_NULLABLE,
			&col.DATA_TYPE,
			&col.CHARACTER_MAXIMUM_LENGTH,
			&col.CHARACTER_OCTET_LENGTH,
			&col.NUMERIC_PRECISION,
			&col.NUMERIC_SCALE,
			&col.DATETIME_PRECISION,
			&col.CHARACTER_SET_NAME,
			&col.COLLATION_NAME,
			&col.COLUMN_TYPE,
			&col.COLUMN_KEY,
			&col.EXTRA,
			&col.PRIVILEGES,
			&col.COLUMN_COMMENT,
			&col.GENERATION_EXPRESSION,
		)
		if err != nil {
			log.Println(err.Error())
			return TableInfo{}
		}

		col.ColumnName = parser.StringToHump(col.COLUMN_NAME)
		col.GoType = TypeForMysqlToGo[col.DATA_TYPE]
		if col.GoType == "int32" || col.GoType == "int64" {
			if strings.Contains(col.COLUMN_TYPE, "unsigned") {
				col.GoType = "u" + col.GoType
			}
		}

		if _, ok := tableColumns[col.TABLE_NAME]; !ok {
			tableColumns[col.TABLE_NAME] = []tableColumn{}
		}
		col.Index = tableIndex[col.TABLE_NAME][col.COLUMN_NAME]
		tableColumns[col.TABLE_NAME] = append(tableColumns[col.TABLE_NAME], col)
	}

	return Filter(tableColumns)
}

// Filter 过滤分表格式
// table_{0-9} 只返回table
func Filter(tableColumns map[string][]tableColumn) TableInfo {
	info := TableInfo{
		Columns: make(map[string][]tableColumn),
		Infos:   make(map[string]TableInfos),
	}
	tableSort := make(map[string]int)
	for tableName, columns := range tableColumns {
		arr := strings.Split(tableName, "_")
		arrLen := len(arr)
		if arrLen > 1 {
			str := arr[arrLen-1]
			tn, err := strconv.Atoi(str)
			if err == nil {
				tableName = strings.ReplaceAll(tableName, "_"+str, "")
				info.Infos[tableName] = TableInfos{
					"sub": "true", // 分表
				}
				// 保留数字最大的
				n, ok := tableSort[tableName]
				if ok && n > tn {
					continue
				}
				tableSort[tableName] = tn
			}
		}
		info.Columns[tableName] = columns
	}
	return info
}

func (d *DB) tableIndex() map[string]map[string][]tableColumnIndex {
	got := make(map[string]map[string][]tableColumnIndex, 0)
	var sqlStr = `SELECT * FROM information_schema.statistics WHERE table_schema = DATABASE ();`
	rows, err := d.db.Query(sqlStr)
	if err != nil {
		log.Println("Error reading table information: ", err.Error())
		return nil
	}
	defer rows.Close()
	columns, _ := rows.Columns()
	length := len(columns)
	for rows.Next() {
		value := make([]interface{}, length)
		columnPointers := make([]interface{}, length)
		for i := 0; i < length; i++ {
			columnPointers[i] = &value[i]
		}
		rows.Scan(columnPointers...)
		data := make(map[string]string)
		for i := 0; i < length; i++ {
			columnName := columns[i]
			columnValue := columnPointers[i].(*interface{})
			if *columnValue != nil {
				data[columnName] = string((*columnValue).([]byte))
			}
		}
		tableIndex, ok := got[data["TABLE_NAME"]]
		if !ok {
			tableIndex = make(map[string][]tableColumnIndex)
		}

		if _, ok := tableIndex[data["COLUMN_NAME"]]; !ok {
			tableIndex[data["COLUMN_NAME"]] = make([]tableColumnIndex, 0)
		}

		tableIndex[data["COLUMN_NAME"]] = append(tableIndex[data["COLUMN_NAME"]], tableColumnIndex{
			COMMENT:     data["COMMENT"],
			INDEX_NAME:  data["INDEX_NAME"],
			CARDINALITY: data["CARDINALITY"],
			INDEX_TYPE:  data["INDEX_TYPE"],
			NON_UNIQUE:  data["NON_UNIQUE"],
		})
		got[data["TABLE_NAME"]] = tableIndex
	}
	return got
}

type TableInfos map[string]interface{}

func (t TableInfos) IsSub() bool {
	if _, ok := t["sub"]; ok {
		return true
	}
	return false
}

type TableInfo struct {
	Columns map[string][]tableColumn
	Infos   map[string]TableInfos
}

type tableColumn struct {
	// 驼峰命名的字段
	ColumnName string
	GoType     string
	mysql
	Index []tableColumnIndex
}

type mysql struct {
	TABLE_CATALOG            string
	TABLE_SCHEMA             string
	TABLE_NAME               string
	COLUMN_NAME              string
	ORDINAL_POSITION         string
	COLUMN_DEFAULT           *string
	IS_NULLABLE              *string
	DATA_TYPE                string
	CHARACTER_MAXIMUM_LENGTH *string
	CHARACTER_OCTET_LENGTH   *string
	NUMERIC_PRECISION        *string
	NUMERIC_SCALE            *string
	DATETIME_PRECISION       *string
	CHARACTER_SET_NAME       *string
	COLLATION_NAME           *string
	COLUMN_TYPE              string
	COLUMN_KEY               string
	EXTRA                    string
	PRIVILEGES               string
	COLUMN_COMMENT           string
	GENERATION_EXPRESSION    *string
}

type tableColumnIndex struct {
	COMMENT     string
	INDEX_NAME  string
	CARDINALITY string
	INDEX_TYPE  string
	NON_UNIQUE  string
}

var TypeForMysqlToGo = map[string]string{
	"int":                "int32",
	"integer":            "int32",
	"tinyint":            "int32",
	"smallint":           "int32",
	"mediumint":          "int32",
	"bigint":             "int64",
	"int unsigned":       "uint32",
	"integer unsigned":   "uint32",
	"tinyint unsigned":   "uint32",
	"smallint unsigned":  "uint32",
	"mediumint unsigned": "uint32",
	"bigint unsigned":    "uint64",
	"bit":                "int64",
	"bool":               "bool",
	"enum":               "string",
	"set":                "string",
	"varchar":            "string",
	"char":               "string",
	"tinytext":           "string",
	"mediumtext":         "string",
	"text":               "string",
	"longtext":           "string",
	"blob":               "string",
	"tinyblob":           "string",
	"mediumblob":         "string",
	"longblob":           "string",
	"date":               "database.Time", // time.Time or string
	"datetime":           "database.Time", // time.Time or string
	"timestamp":          "database.Time", // time.Time or string
	"time":               "database.Time", // time.Time or string
	"float":              "float64",
	"double":             "float64",
	"decimal":            "float64",
	"binary":             "string",
	"varbinary":          "string",
	"json":               "database.JSON",
}

func NewDb(conf map[interface{}]interface{}) *DB {
	config := services.NewConfig(conf)
	db, err := sql.Open("mysql", fmt.Sprintf(
		"%s:%s@tcp(%s)/%s?interpolateParams=true",
		config.GetString("username", "root"),
		config.GetString("password", "123456"),
		config.GetString("host", "localhost")+":"+config.GetString("port", "3306"),
		config.GetString("database", "demo"),
	))
	if err != nil {
		panic(err)
	}
	// See "Important settings" section.
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	return &DB{
		db: db,
	}
}
