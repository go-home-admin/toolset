package orm

import (
	"github.com/go-home-admin/home/bootstrap/services/database"
	"github.com/go-home-admin/toolset/parser"
	"log"
	"os"
	"strings"
)

func GenSqlite(table string, goType parser.GoType, out string) {
	table = parser.StringToSnake(table)

	file := out + "/z_" + table

	if _, err := os.Stat(file + "_lock.go"); !os.IsNotExist(err) {
		return
	}

	imports := parser.GetImportStrForMap(map[string]string{
		"strings":                           "",
		"gorm.io/gorm":                      "gorm",
		"github.com/sirupsen/logrus":        "logrus",
		"database/sql":                      "sql",
		"github.com/go-home-admin/home/app": "home",
	})

	columns := []tableColumn{}
	for _, attrName := range goType.AttrsSort {
		attr := goType.Attrs[attrName]
		columns = append(columns, tableColumn{
			ColumnName: attr.Name,
			GoType:     attr.TypeName,
			mysql: mysql{

				TABLE_NAME:  attr.Name,
				COLUMN_NAME: attr.Name,

				IS_NULLABLE: database.StrPointer("NO"),
				DATA_TYPE:   attr.TypeName,
				COLUMN_TYPE: attr.TypeName,
			},
		})
	}

	name := "sqlite"

	str := "package " + name
	str += "\nimport (" + imports + "\n)"
	str += "\n" //  + genOrmStruct(table, columns, Conf{}, nil)

	var baseFunStr string
	baseFunStr = strings.ReplaceAll(baseMysqlFuncStr,
		"tx = providers.GetBean(\"mysql, {db}\").(*gorm.DB).Model(&MysqlTableName{})",
		"tx = DB()")
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
