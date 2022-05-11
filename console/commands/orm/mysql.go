package orm

import (
	"database/sql"
	_ "embed"
	"fmt"
	"github.com/go-home-admin/home/bootstrap/services"
	"github.com/go-home-admin/toolset/parser"
	_ "github.com/go-sql-driver/mysql"
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

func GenMysql(name string, conf map[interface{}]interface{}, out string) {
	if !IsExist(out) {
		os.MkdirAll(out, 0766)
	}

	db := NewDb(conf)
	tableColumns := db.tableColumns()

	// 计算import
	imports := getImports(tableColumns)
	for table, columns := range tableColumns {
		mysqlTableName := parser.StringToSnake(table)
		file := out + "/" + mysqlTableName

		str := "package " + name
		str += "\nimport (" + imports[table] + "\n)"
		str += "\n" + genOrmStruct(table, columns)

		baseFunStr := baseMysqlFuncStr
		for old, new := range map[string]string{
			"MysqlTableName": parser.StringToHump(table),
			"{table_name}":   table,
			"{db}":           name,
		} {
			baseFunStr = strings.ReplaceAll(baseFunStr, old, new)
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
		if strInStr(column.Field, []string{"id", "code"}) {
			str += "\nfunc (l " + TableName + "List) Get" + column.ColumnName + "List() []" + column.GoaType + " {" +
				"\n\tgot := make([]" + column.GoaType + ", 0)\n\tfor _, val := range l {" +
				"\n\t\tgot = append(got, val." + column.ColumnName + ")" +
				"\n\t}" +
				"\n\treturn got" +
				"\n}"

			str += "\nfunc (l " + TableName + "List) Get" + column.ColumnName + "Map() map[" + column.GoaType + "]*" + TableName + " {" +
				"\n\tgot := make(map[" + column.GoaType + "]*" + TableName + ")\n\tfor _, val := range l {" +
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
		str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "(val " + column.GoaType + ") *Orm" + TableName + " {" +
			"\n\torm.db.Where(\"`" + column.Field + "` = ?\", val)" +
			"\n\treturn orm" +
			"\n}"

		if column.PrimaryKey != "" {
			// if 主键, 生成In, > <
			str += "\nfunc (orm *Orm" + TableName + ") InsertGet" + column.ColumnName + "(row *" + TableName + ") " + column.GoaType + " {" +
				"\n\torm.db.Create(row)" +
				"\n\treturn row." + column.ColumnName +
				"\n}"

			str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "In(val []" + column.GoaType + ") *Orm" + TableName + " {" +
				"\n\torm.db.Where(\"`" + column.Field + "` IN ?\", val)" +
				"\n\treturn orm" +
				"\n}"

			str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "Gt(val " + column.GoaType + ") *Orm" + TableName + " {" +
				"\n\torm.db.Where(\"`" + column.Field + "` > ?\", val)" +
				"\n\treturn orm" +
				"\n}"
			str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "Gte(val " + column.GoaType + ") *Orm" + TableName + " {" +
				"\n\torm.db.Where(\"`" + column.Field + "` >= ?\", val)" +
				"\n\treturn orm" +
				"\n}"

			str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "Lt(val " + column.GoaType + ") *Orm" + TableName + " {" +
				"\n\torm.db.Where(\"`" + column.Field + "` < ?\", val)" +
				"\n\treturn orm" +
				"\n}"
			str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "Lte(val " + column.GoaType + ") *Orm" + TableName + " {" +
				"\n\torm.db.Where(\"`" + column.Field + "` <= ?\", val)" +
				"\n\treturn orm" +
				"\n}"
		} else {
			// 索引，或者枚举字段
			if strInStr(column.Field, []string{"id", "code", "status", "state"}) {
				// else if 名称存在 id, code, status 生成in操作
				str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "In(val []" + column.GoaType + ") *Orm" + TableName + " {" +
					"\n\torm.db.Where(\"`" + column.Field + "` IN ?\", val)" +
					"\n\treturn orm" +
					"\n}"

				str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "Ne(val " + column.GoaType + ") *Orm" + TableName + " {" +
					"\n\torm.db.Where(\"`" + column.Field + "` <> ?\", val)" +
					"\n\treturn orm" +
					"\n}"
			}
			// 时间字段
			if strInStr(column.Field, []string{"created", "updated", "time", "_at"}) || (column.GoaType == "database.Time") {
				str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "Between(begin " + column.GoaType + ", end " + column.GoaType + ") *Orm" + TableName + " {" +
					"\n\torm.db.Where(\"`" + column.Field + "` BETWEEN ? AND ?\", begin, end)" +
					"\n\treturn orm" +
					"\n}"

				str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "Lte(val " + column.GoaType + ") *Orm" + TableName + " {" +
					"\n\torm.db.Where(\"`" + column.Field + "` <= ?\", val)" +
					"\n\treturn orm" +
					"\n}"

				str += "\nfunc (orm *Orm" + TableName + ") Where" + column.ColumnName + "Gte(val " + column.GoaType + ") *Orm" + TableName + " {" +
					"\n\torm.db.Where(\"`" + column.Field + "` >= ?\", val)" +
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

//go:embed mysql.go.text
var baseMysqlFuncStr string

// 字段类型引入
var alias = map[string]string{
	"database": "github.com/go-home-admin/home/bootstrap/services/database",
}

// 获得 table => map{alias => github.com/*}
func getImports(tableColumns map[string][]tableColumn) map[string]string {
	got := make(map[string]string)
	for table, columns := range tableColumns {
		// 初始引入
		tm := map[string]string{
			"gorm.io/gorm": "gorm",
			"github.com/go-home-admin/home/bootstrap/services/app": "app",
			"github.com/sirupsen/logrus":                           "logrus",
			"database/sql":                                         "sql",
		}
		for _, column := range columns {
			index := strings.Index(column.GoaType, ".")
			if index != -1 {
				as := column.GoaType[:index]
				importStr := alias[as]
				tm[importStr] = as
			}
		}
		got[table] = parser.GetImportStrForMap(tm)
	}

	return got
}

func genOrmStruct(table string, columns []tableColumn) string {
	TableName := parser.StringToHump(table)

	str := `type {TableName} struct {`
	for _, column := range columns {
		str += "\n\t" + parser.StringToHump(column.Field) + " " + column.GoaType +
			"`" + genGormTag(column) + "` // " +
			strings.ReplaceAll(column.ColumnComment, "\n", " ")
	}

	str = strings.ReplaceAll(str, "{TableName}", TableName)
	return "\n" + str + "\n}"
}

func genGormTag(column tableColumn) string {
	var arr []string
	if column.PrimaryKey != "" {
		arr = append(arr, "primaryKey")
	}

	arr = append(arr, "column:"+column.Field)
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
	var sqlStr = `SELECT
	COLUMN_NAME,
	DATA_TYPE,
	IS_NULLABLE,
	TABLE_NAME,
	COLUMN_COMMENT,
    COLUMN_KEY
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
		return nil
	}

	defer rows.Close()
	tableColumns := make(map[string][]tableColumn)
	for rows.Next() {
		col := tableColumn{}
		err = rows.Scan(&col.ColumnName, &col.MysqlType, &col.Nullable, &col.TableName, &col.ColumnComment, &col.PrimaryKey)
		if err != nil {
			log.Println(err.Error())
			return nil
		}

		col.Field = col.ColumnName
		col.ColumnName = parser.StringToHump(col.ColumnName)
		col.GoaType = typeForMysqlToGo[col.MysqlType]

		if _, ok := tableColumns[col.TableName]; !ok {
			tableColumns[col.TableName] = []tableColumn{}
		}
		tableColumns[col.TableName] = append(tableColumns[col.TableName], col)
	}
	return tableColumns
}

type tableColumn struct {
	PrimaryKey    string
	ColumnName    string
	GoaType       string
	MysqlType     string
	Nullable      string
	TableName     string
	ColumnComment string
	Field         string
}

var typeForMysqlToGo = map[string]string{
	"int":                "int64",
	"integer":            "int64",
	"tinyint":            "int32",
	"smallint":           "int32",
	"mediumint":          "int64",
	"bigint":             "int64",
	"int unsigned":       "uint64",
	"integer unsigned":   "uint64",
	"tinyint unsigned":   "uint32",
	"smallint unsigned":  "uint64",
	"mediumint unsigned": "uint64",
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
	"json":               "database.Json",
}

func NewDb(conf map[interface{}]interface{}) *DB {
	config := services.NewConfig(conf)
	db, err := sql.Open("mysql", fmt.Sprintf(
		"%s:%s@tcp(%s)/%s",
		config.GetString("username", "root"),
		config.GetString("password", "123456"),
		config.GetString("host", "localhost:"+config.GetString("port", "3306")),
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
