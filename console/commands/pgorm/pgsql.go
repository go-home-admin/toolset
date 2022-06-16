package pgorm

import (
	"database/sql"
	_ "embed"
	"fmt"
	"github.com/go-home-admin/home/bootstrap/services"
	"github.com/go-home-admin/toolset/parser"
	_ "github.com/lib/pq"
	"log"
	"os"
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
	tableColumns := db.tableColumns()

	// 计算import
	imports := getImports(tableColumns)
	for table, columns := range tableColumns {
		tableName := parser.StringToSnake(table)
		file := out + "/" + tableName

		str := "package " + name
		str += "\nimport (" + imports[table] + "\n)"
		str += "\n" + genOrmStruct(table, columns, conf)

		baseFunStr := baseMysqlFuncStr
		for old, newStr := range map[string]string{
			"{orm_table_name}": parser.StringToHump(table),
			"{table_name}":     table,
			"{db}":             name,
		} {
			baseFunStr = strings.ReplaceAll(baseFunStr, old, newStr)
		}

		str += baseFunStr
		str += genFieldFunc(table, columns)
		str += genListFunc(table, columns)
		str += genWithFunc(table, columns, conf)
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
		if strInStr(column.ColumnName, []string{"id", "code"}) {
			goType := column.GoType
			if column.IsNullable {
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

func genWithFunc(table string, columns []tableColumn, conf Conf) string {
	TableName := parser.StringToHump(table)
	str := ""
	if helper, ok := conf["helper"]; ok {
		helperConf := helper.(map[interface{}]interface{})
		tableConfig, ok := helperConf[table].([]interface{})
		if ok {
			for _, c := range tableConfig {
				cf := c.(map[interface{}]interface{})
				with := cf["with"]
				tbName := parser.StringToHump(cf["table"].(string))
				switch with {
				case "many2many":

				default:
					str += "\nfunc (orm *Orm" + TableName + ") Joins" + tbName + "(args ...interface{}) *Orm" + TableName + " {" +
						"\n\torm.db.Joins(\"" + cf["alias"].(string) + "\", args...)" +
						"\n\treturn orm" +
						"\n}"
					str += "\nfunc (orm *Orm" + TableName + ") Preload" + tbName + "(args ...interface{}) *Orm" + TableName + " {" +
						"\n\torm.db.Preload(\"" + cf["alias"].(string) + "\", args...)" +
						"\n\treturn orm" +
						"\n}"
				}
			}
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
			"\n\torm.db.Where(\"`" + column.ColumnName + "` = ?\", val)" +
			"\n\treturn orm" +
			"\n}"

		if column.IsPKey {
			// if 主键, 生成In, > <
			str += "\nfunc (orm *Orm" + TableName + ") InsertGet" + column.ColumnName + "(row *" + TableName + ") " + column.GoType + " {" +
				"\n\torm.db.Create(row)" +
				"\n\treturn row." + column.ColumnName +
				"\n}"

			str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "In(val []" + column.GoType + ") *Orm" + TableName + " {" +
				"\n\torm.db.Where(\"`" + column.ColumnName + "` IN ?\", val)" +
				"\n\treturn orm" +
				"\n}"

			str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "Gt(val " + column.GoType + ") *Orm" + TableName + " {" +
				"\n\torm.db.Where(\"`" + column.ColumnName + "` > ?\", val)" +
				"\n\treturn orm" +
				"\n}"
			str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "Gte(val " + column.GoType + ") *Orm" + TableName + " {" +
				"\n\torm.db.Where(\"`" + column.ColumnName + "` >= ?\", val)" +
				"\n\treturn orm" +
				"\n}"

			str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "Lt(val " + column.GoType + ") *Orm" + TableName + " {" +
				"\n\torm.db.Where(\"`" + column.ColumnName + "` < ?\", val)" +
				"\n\treturn orm" +
				"\n}"
			str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "Lte(val " + column.GoType + ") *Orm" + TableName + " {" +
				"\n\torm.db.Where(\"`" + column.ColumnName + "` <= ?\", val)" +
				"\n\treturn orm" +
				"\n}"
		} else {
			// 索引，或者枚举字段
			if strInStr(column.ColumnName, []string{"id", "code", "status", "state"}) {
				// else if 名称存在 id, code, status 生成in操作
				str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "In(val []" + column.GoType + ") *Orm" + TableName + " {" +
					"\n\torm.db.Where(\"`" + column.ColumnName + "` IN ?\", val)" +
					"\n\treturn orm" +
					"\n}"

				str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "Ne(val " + column.GoType + ") *Orm" + TableName + " {" +
					"\n\torm.db.Where(\"`" + column.ColumnName + "` <> ?\", val)" +
					"\n\treturn orm" +
					"\n}"
			}
			// 时间字段
			if strInStr(column.ColumnName, []string{"created", "updated", "time", "_at"}) || (column.GoType == "database.Time") {
				str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "Between(begin " + column.GoType + ", end " + column.GoType + ") *Orm" + TableName + " {" +
					"\n\torm.db.Where(\"`" + column.ColumnName + "` BETWEEN ? AND ?\", begin, end)" +
					"\n\treturn orm" +
					"\n}"

				str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "Lte(val " + column.GoType + ") *Orm" + TableName + " {" +
					"\n\torm.db.Where(\"`" + column.ColumnName + "` <= ?\", val)" +
					"\n\treturn orm" +
					"\n}"

				str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "Gte(val " + column.GoType + ") *Orm" + TableName + " {" +
					"\n\torm.db.Where(\"`" + column.ColumnName + "` >= ?\", val)" +
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

//go:embed pgsql.go.text
var baseMysqlFuncStr string

// 字段类型引入
var alias = map[string]string{
	"database":  "github.com/go-home-admin/home/bootstrap/services/database",
	"datatypes": "gorm.io/datatypes",
}

// 获得 table => map{alias => github.com/*}
func getImports(tableColumns map[string][]tableColumn) map[string]string {
	got := make(map[string]string)
	for table, columns := range tableColumns {
		// 初始引入
		tm := map[string]string{
			"gorm.io/gorm": "gorm",
			"github.com/go-home-admin/home/bootstrap/providers": "providers",
			"github.com/sirupsen/logrus":                        "logrus",
			"database/sql":                                      "sql",
		}
		for _, column := range columns {
			index := strings.Index(column.GoType, ".")
			if index != -1 && column.GoType[:index] != "gorm" {
				as := strings.Replace(column.GoType[:index], "*", "", 1)
				importStr := alias[as]
				tm[importStr] = as
			}
		}
		got[table] = parser.GetImportStrForMap(tm)
	}

	return got
}

func genOrmStruct(table string, columns []tableColumn, conf Conf) string {
	TableName := parser.StringToHump(table)

	hasField := make(map[string]bool)
	str := `type {TableName} struct {`
	for _, column := range columns {
		p := ""
		if column.IsNullable {
			p = "*"
		}
		hasField[column.ColumnName] = true
		fieldName := parser.StringToHump(column.ColumnName)
		str += fmt.Sprintf("\n\t%v %v%v`%v` // %v", fieldName, p, column.GoType, genGormTag(column), strings.ReplaceAll(column.Comment, "\n", " "))
	}
	// 表依赖
	if helper, ok := conf["helper"]; ok {
		helperConf := helper.(map[interface{}]interface{})
		tableConfig, ok := helperConf[table].([]interface{})
		if ok {
			for _, c := range tableConfig {
				cf := c.(map[interface{}]interface{})
				with := cf["with"]
				tbName := parser.StringToHump(cf["table"].(string))
				switch with {
				case "belongs_to":
					str += fmt.Sprintf("\n\t%v *%v `gorm:\"%v\"`", parser.StringToHump(cf["alias"].(string)), tbName, cf["gorm"])
				case "has_one":
					str += fmt.Sprintf("\n\t%v *%v `gorm:\"%v\"`", parser.StringToHump(cf["alias"].(string)), tbName, cf["gorm"])
				case "has_many":
					str += fmt.Sprintf("\n\t%v []%v `gorm:\"%v\"`", parser.StringToHump(cf["alias"].(string)), tbName, cf["gorm"])
				case "many2many":
					str += fmt.Sprintf("\n\t%v []%v `gorm:\"%v\"`", parser.StringToHump(cf["alias"].(string)), tbName, cf["gorm"])
				default:
					panic("with: belongs_to,has_one,has_many,many2many")
				}
			}
		}
	}

	str = strings.ReplaceAll(str, "{TableName}", TableName)
	return "\n" + str + "\n}"
}

func genGormTag(column tableColumn) string {
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
	}
	// default
	if column.ColumnDefault != "" {
		arr = append(arr, "default:"+column.ColumnDefault)
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

func (d *DB) tableColumns() map[string][]tableColumn {
	var sqlStr = "SELECT tablename FROM pg_tables WHERE schemaname = 'public'"

	rows, err := d.db.Query(sqlStr)
	if err != nil {
		log.Println("Error reading table information: ", err.Error())
		return nil
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
SELECT i.column_name, i.column_default, i.is_nullable, i.udt_name, col_description(a.attrelid,a.attnum) as comment
FROM information_schema.columns as i 
LEFT JOIN pg_class as c on c.relname = i.table_name
LEFT JOIN pg_attribute as a on a.attrelid = c.oid and a.attname = i.column_name
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
		for _rows.Next() {
			var (
				column_name    string
				column_default *string
				is_nullable    string
				udt_name       string
				comment        *string
			)
			err = _rows.Scan(&column_name, &column_default, &is_nullable, &udt_name, &comment)
			if err != nil {
				panic(err)
			}
			var columnComment string
			if comment != nil {
				columnComment = *comment
			}
			var ColumnDefault string
			if column_default != nil {
				ColumnDefault = *column_default
			}

			ormColumns[tableName] = append(ormColumns[tableName], tableColumn{
				ColumnName:    parser.StringToHump(column_name),
				ColumnDefault: ColumnDefault,
				PgType:        udt_name,
				GoType:        PgTypeToGoType(udt_name, column_name),
				IsNullable:    is_nullable == "YES",
				IsPKey:        false,
				Comment:       columnComment,
			})
		}
	}
	return ormColumns
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
}

func PgTypeToGoType(pgType string, columnName string) string {
	switch pgType {
	case "int2", "int4":
		return "int32"
	case "int8":
		return "int64"
	case "date":
		return "datatypes.Date"
	case "json", "jsonb":
		return "database.JSON"
	case "time", "timetz":
		return "database.Time"
	case "numeric":
		return "float64"
	default:
		if strings.Contains(pgType, "timestamp") {
			if columnName == "deleted_at" {
				return "gorm.DeletedAt"
			} else {
				return "database.Time"
			}
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
