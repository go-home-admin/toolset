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

type Conf map[interface{}]interface{}

func GenMysql(name string, conf Conf, out string) {
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
		str += "\n" + genOrmStruct(table, columns, conf)

		baseFunStr := baseMysqlFuncStr
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
		if strInStr(column.COLUMN_NAME, []string{"id", "code"}) {
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
			"\n\torm.db.Where(\"`" + column.COLUMN_NAME + "` = ?\", val)" +
			"\n\treturn orm" +
			"\n}"

		if column.COLUMN_KEY != "" {
			goType := column.GoType
			if *column.IS_NULLABLE == "YES" {
				goType = "*" + goType
			}
			// if 主键, 生成In, > <
			str += "\nfunc (orm *Orm" + TableName + ") InsertGet" + column.ColumnName + "(row *" + TableName + ") " + goType + " {" +
				"\n\torm.db.Create(row)" +
				"\n\treturn row." + column.ColumnName +
				"\n}"

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
			"github.com/go-home-admin/home/bootstrap/providers": "providers",
			"github.com/sirupsen/logrus":                        "logrus",
			"database/sql":                                      "sql",
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

func genOrmStruct(table string, columns []tableColumn, conf Conf) string {
	TableName := parser.StringToHump(table)

	hasField := make(map[string]bool)
	str := `type {TableName} struct {`
	for _, column := range columns {
		p := ""
		if *column.IS_NULLABLE == "YES" {
			p = "*"
		}
		hasField[column.COLUMN_NAME] = true
		fieldName := parser.StringToHump(column.COLUMN_NAME)
		str += fmt.Sprintf("\n\t%v %v%v`%v` // %v", fieldName, p, column.GoType, genGormTag(column), strings.ReplaceAll(column.COLUMN_COMMENT, "\n", " "))
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

func (d *DB) tableColumns() map[string][]tableColumn {
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
		return nil
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
			return nil
		}

		col.ColumnName = parser.StringToHump(col.COLUMN_NAME)
		col.GoType = typeForMysqlToGo[col.DATA_TYPE]

		if _, ok := tableColumns[col.TABLE_NAME]; !ok {
			tableColumns[col.TABLE_NAME] = []tableColumn{}
		}
		col.Index = tableIndex[col.TABLE_NAME][col.COLUMN_NAME]
		tableColumns[col.TABLE_NAME] = append(tableColumns[col.TABLE_NAME], col)
	}

	return tableColumns
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

var typeForMysqlToGo = map[string]string{
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
