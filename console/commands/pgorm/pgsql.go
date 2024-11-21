package pgorm

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/go-home-admin/home/bootstrap/services"
	"github.com/go-home-admin/toolset/console/commands/orm"
	"github.com/go-home-admin/toolset/parser"
	_ "github.com/lib/pq"
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

func GenSql(name string, conf Conf, out string) {
	if !IsExist(out) {
		os.MkdirAll(out, 0766)
	}

	db := NewDb(conf)
	tableInfos := db.tableColumns()
	tableColumns := tableInfos.Columns

	root, _ := os.Getwd()
	file, err := os.ReadFile(root + "/config/database/" + name + ".json")
	var relationship map[string][]*orm.Relationship
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
			baseFunStr = basePgsqlSubInfoStr
		} else {
			baseFunStr = basePgsqlFuncStr
		}
		for old, newStr := range map[string]string{
			"{orm_table_name}": parser.StringToHump(table),
			"{table_name}":     table,
			"{connect_name}":   name,
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
		if column.GoType == "[]byte" {
			continue
		}
		ColumnName := parser.StringToHump(column.ColumnName)
		// 索引，或者枚举字段
		if strInStr(column.ColumnName, []string{"id", "code"}) || strInStr(column.Comment, []string{"@index"}) {
			goType := column.GoType
			if column.IsNullable {
				goType = "*" + goType
			}
			str += "\nfunc (l " + TableName + "List) Get" + ColumnName + "List() []" + goType + " {" +
				"\n\tgot := make([]" + goType + ", 0)\n\tfor _, val := range l {" +
				"\n\t\tgot = append(got, val." + ColumnName + ")" +
				"\n\t}" +
				"\n\treturn got" +
				"\n}"

			str += "\nfunc (l " + TableName + "List) Get" + ColumnName + "Map() map[" + goType + "]*" + TableName + " {" +
				"\n\tgot := make(map[" + goType + "]*" + TableName + ")\n\tfor _, val := range l {" +
				"\n\t\tgot[val." + ColumnName + "] = val" +
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
		ColumnName := parser.StringToHump(column.ColumnName)
		// 等于函数
		str += "\nfunc (orm *Orm" + TableName + ") Where" + ColumnName + "(val " + column.GoType + ") *Orm" + TableName + " {" +
			"\n\torm.db.Where(\"\\\"" + column.ColumnName + "\\\" = ?\", val)" +
			"\n\treturn orm" +
			"\n}"

		if strInStr(column.GoType, []string{"int32", "int64"}) {
			goType := column.GoType
			if column.IsNullable {
				goType = "*" + goType
			}
			// if 主键, 生成In, > <
			if column.IsPKey {
				str += "\nfunc (orm *Orm" + TableName + ") InsertGet" + ColumnName + "(row *" + TableName + ") " + goType + " {" +
					"\n\torm.db.Create(row)" +
					"\n\treturn row." + ColumnName +
					"\n}"
			}

			str += "\nfunc (orm *Orm" + TableName + ") Where" + ColumnName + "In(val []" + column.GoType + ") *Orm" + TableName + " {" +
				"\n\torm.db.Where(\"\\\"" + column.ColumnName + "\\\" IN ?\", val)" +
				"\n\treturn orm" +
				"\n}"

			str += "\nfunc (orm *Orm" + TableName + ") Where" + ColumnName + "Gt(val " + column.GoType + ") *Orm" + TableName + " {" +
				"\n\torm.db.Where(\"\\\"" + column.ColumnName + "\\\" > ?\", val)" +
				"\n\treturn orm" +
				"\n}"
			str += "\nfunc (orm *Orm" + TableName + ") Where" + ColumnName + "Gte(val " + column.GoType + ") *Orm" + TableName + " {" +
				"\n\torm.db.Where(\"\\\"" + column.ColumnName + "\\\" >= ?\", val)" +
				"\n\treturn orm" +
				"\n}"

			str += "\nfunc (orm *Orm" + TableName + ") Where" + ColumnName + "Lt(val " + column.GoType + ") *Orm" + TableName + " {" +
				"\n\torm.db.Where(\"\\\"" + column.ColumnName + "\\\" < ?\", val)" +
				"\n\treturn orm" +
				"\n}"
			str += "\nfunc (orm *Orm" + TableName + ") Where" + ColumnName + "Lte(val " + column.GoType + ") *Orm" + TableName + " {" +
				"\n\torm.db.Where(\"\\\"" + column.ColumnName + "\\\" <= ?\", val)" +
				"\n\treturn orm" +
				"\n}"
		} else {
			// 索引，或者枚举字段
			if strInStr(column.ColumnName, []string{"id", "code", "status", "state"}) {
				// else if 名称存在 id, code, status 生成in操作
				str += "\nfunc (orm *Orm" + TableName + ") Where" + ColumnName + "In(val []" + column.GoType + ") *Orm" + TableName + " {" +
					"\n\torm.db.Where(\"\\\"" + column.ColumnName + "\\\" IN ?\", val)" +
					"\n\treturn orm" +
					"\n}"

				str += "\nfunc (orm *Orm" + TableName + ") Where" + ColumnName + "Ne(val " + column.GoType + ") *Orm" + TableName + " {" +
					"\n\torm.db.Where(\"\\\"" + column.ColumnName + "\\\" <> ?\", val)" +
					"\n\treturn orm" +
					"\n}"
			}
			// 时间字段
			if strInStr(column.ColumnName, []string{"created", "updated", "time", "_at"}) || (column.GoType == "database.Time") {
				str += "\nfunc (orm *Orm" + TableName + ") Where" + ColumnName + "Between(begin " + column.GoType + ", end " + column.GoType + ") *Orm" + TableName + " {" +
					"\n\torm.db.Where(\"\\\"" + column.ColumnName + "\\\" BETWEEN ? AND ?\", begin, end)" +
					"\n\treturn orm" +
					"\n}"

				str += "\nfunc (orm *Orm" + TableName + ") Where" + ColumnName + "Lte(val " + column.GoType + ") *Orm" + TableName + " {" +
					"\n\torm.db.Where(\"\\\"" + column.ColumnName + "\\\" <= ?\", val)" +
					"\n\treturn orm" +
					"\n}"

				str += "\nfunc (orm *Orm" + TableName + ") Where" + ColumnName + "Gte(val " + column.GoType + ") *Orm" + TableName + " {" +
					"\n\torm.db.Where(\"\\\"" + column.ColumnName + "\\\" >= ?\", val)" +
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

//go:embed pgsql.go.subtext
var basePgsqlSubInfoStr string

//go:embed pgsql.go.text
var basePgsqlFuncStr string

// 字段类型引入
var alias = map[string]string{
	"database": "github.com/go-home-admin/home/bootstrap/services/database",
}

// 获得 table => map{alias => github.com/*}
func getImports(infos map[string]orm.TableInfos, tableColumns map[string][]tableColumn) map[string]string {
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

func genOrmStruct(table string, columns []tableColumn, conf Conf, relationships []*orm.Relationship) string {
	TableName := parser.StringToHump(table)
	config := services.NewConfig(conf)
	deletedField := config.GetString("deleted_field")
	hasField := make(map[string]bool)
	str := `type {TableName} struct {`
	for _, column := range columns {
		p := ""
		if column.IsNullable && !(column.ColumnName == "deleted_at" && column.GoType == "database.Time") && column.PgType != "bytea" {
			p = "*"
		}
		if column.GoType == "database.Time" && (column.ColumnName == deletedField || (deletedField == "" && column.ColumnName == "deleted_at")) {
			column.GoType = "gorm.DeletedAt"
		}

		// 使用注释@Type(int), 强制设置生成的go struct 属性 类型
		if i := strings.Index(column.ColumnName, "@type("); i != -1 {
			s := column.Comment[i+6:]
			e := strings.Index(s, ")")
			column.GoType = s[:e]
		}

		hasField[column.ColumnName] = true
		fieldName := parser.StringToHump(column.ColumnName)
		str += fmt.Sprintf("\n\t%v %v%v`%v` // %v", fieldName, p, column.GoType, genGormTag(column, conf), strings.ReplaceAll(column.Comment, "\n", " "))
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
		str += fmt.Sprintf("{TableName}Field%s = \"%s\"\n", parser.StringToHump(column.ColumnName), column.ColumnName)
	}
	str += ")"

	str = strings.ReplaceAll(str, "{TableName}", TableName)
	return "\n" + str + "\n"
}

func genGormTag(column tableColumn, conf Conf) string {
	var arr []string
	// 字段
	arr = append(arr, "column:"+column.ColumnName)
	if column.ColumnDefault == "CURRENT_TIMESTAMP" {
		arr = append(arr, "autoUpdateTime")
	}
	if strings.Contains(column.ColumnDefault, "nextval") {
		arr = append(arr, "autoIncrement")
	}
	// 类型ing
	arr = append(arr, "type:"+column.PgType)
	// 主键
	if column.IsPKey {
		arr = append(arr, "primaryKey")
	} else if column.IndexName != "" {
		arr = append(arr, "index:"+column.ColumnName)
	}
	// default
	if column.ColumnDefault != "" && !strings.Contains(column.ColumnDefault, "::") {
		arr = append(arr, "default:"+column.ColumnDefault)
	}
	// created_at & updated_at
	if field, ok := conf["created_field"]; ok && field == column.ColumnName {
		arr = append(arr, "autoCreateTime")
	}
	if field, ok := conf["updated_field"]; ok && field == column.ColumnName {
		arr = append(arr, "autoUpdateTime")
	}
	if column.Comment != "" {
		arr = append(arr, fmt.Sprintf("comment:'%v'", strings.ReplaceAll(column.Comment, "'", "")))
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
	var sqlStr = "SELECT tablename FROM pg_tables WHERE schemaname = 'public'"

	rows, err := d.db.Query(sqlStr)
	if err != nil {
		log.Println("Error reading table information: ", err.Error())
		return TableInfo{}
	}
	defer rows.Close()
	ormColumns := make(map[string][]tableColumn)
	for rows.Next() {
		var tableName string
		var pkey string
		_ = rows.Scan(
			&tableName,
		)
		_rows, _ := d.db.Query(`
SELECT i.column_name, i.column_default, i.is_nullable, i.udt_name, col_description(a.attrelid,a.attnum) as comment, ixc.relname
FROM information_schema.columns as i 
LEFT JOIN pg_class as c on c.relname = i.table_name
LEFT JOIN pg_attribute as a on a.attrelid = c.oid and a.attname = i.column_name
LEFT JOIN pg_index ix ON c.oid = ix.indrelid AND a.attnum = ANY(ix.indkey)
LEFT JOIN pg_class ixc ON ixc.oid = ix.indexrelid
WHERE table_schema = 'public' and i.table_name = $1;
		`, tableName)
		defer _rows.Close()
		//获取主键
		__rows, _ := d.db.Query(`
SELECT pg_attribute.attname
FROM pg_constraint
INNER JOIN pg_class ON pg_constraint.conrelid = pg_class.oid
INNER JOIN pg_attribute ON pg_attribute.attrelid = pg_class.oid
AND pg_attribute.attnum = pg_constraint.conkey [ 1 ]
INNER JOIN pg_type ON pg_type.oid = pg_attribute.atttypid
WHERE pg_class.relname = $1 AND pg_constraint.contype = 'p'
		`, tableName)
		defer __rows.Close()
		for __rows.Next() {
			_ = __rows.Scan(&pkey)
		}
		repeatName := map[string]int{}
		for _rows.Next() {
			var (
				column_name    string
				column_default *string
				is_nullable    string
				udt_name       string
				comment        *string
				index_name     *string
			)
			err = _rows.Scan(&column_name, &column_default, &is_nullable, &udt_name, &comment, &index_name)
			if err != nil {
				panic(err)
			}
			if _, ok := repeatName[column_name]; ok {
				continue
			} else {
				repeatName[column_name] = 1
			}
			var columnComment, indexName string
			if comment != nil {
				columnComment = *comment
			}
			if index_name != nil {
				indexName = *index_name
			}
			var ColumnDefault string
			if column_default != nil {
				ColumnDefault = *column_default
			}

			ormColumns[tableName] = append(ormColumns[tableName], tableColumn{
				ColumnName:    column_name,
				ColumnDefault: ColumnDefault,
				PgType:        udt_name,
				GoType:        PgTypeToGoType(udt_name, column_name),
				IsNullable:    is_nullable == "YES",
				IsPKey:        column_name == pkey,
				Comment:       columnComment,
				IndexName:     indexName,
			})
		}
	}
	return Filter(ormColumns)
}

// Filter 过滤分表格式
// table_{0-9} 只返回table
func Filter(tableColumns map[string][]tableColumn) TableInfo {
	info := TableInfo{
		Columns: make(map[string][]tableColumn),
		Infos:   make(map[string]orm.TableInfos),
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
				info.Infos[tableName] = orm.TableInfos{
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

type TableInfo struct {
	Columns map[string][]tableColumn
	Infos   map[string]orm.TableInfos
}

type tableColumn struct {
	// 驼峰命名的字段
	ColumnName    string
	ColumnDefault string
	PgType        string
	GoType        string
	IsNullable    bool
	IsPKey        bool
	Comment       string
	IndexName     string
}

func PgTypeToGoType(pgType string, columnName string) string {
	switch pgType {
	case "int2", "int4":
		return "int32"
	case "int8":
		return "int64"
	case "date":
		return "database.Time"
	case "json", "jsonb":
		return "database.JSON"
	case "time", "timetz":
		return "database.Time"
	case "float4":
		return "float32"
	case "float8", "numeric":
		return "float64"
	case "bool":
		return "bool"
	case "bytea":
		return "[]byte"
	default:
		if strings.Contains(pgType, "timestamp") {
			return "database.Time"
		}
		return "string"
	}
}

func NewDb(conf map[interface{}]interface{}) *DB {
	config := services.NewConfig(conf)
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		config.GetString("username", "root"),
		config.GetString("password", "123456"),
		config.GetString("host", "localhost:"),
		config.GetInt("port", 5432),
		config.GetString("database", "demo"),
	)
	db, err := sql.Open("postgres", connStr)
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
