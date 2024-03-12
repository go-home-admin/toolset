package commands

import (
	"github.com/ctfang/command"
	"github.com/go-home-admin/toolset/console/commands/orm"
	"github.com/go-home-admin/toolset/parser"
	"strings"
)

// SqliteCommand @Bean
type SqliteCommand struct{}

func (SqliteCommand) Configure() command.Configure {
	return command.Configure{
		Name:        "make:sqlite",
		Description: "根据当前目录的文件@Sqlite注释的 Struct生成orm源码",
		Input: command.Argument{
			Option: []command.ArgParam{
				{
					Name:        "scan",
					Description: "扫码目录下的源码; shell(pwd)",
					Default:     "@root",
				},
			},
		},
	}
}

func (SqliteCommand) Execute(input command.Input) {
	root := getRootPath()
	scan := input.GetOption("scan")
	scan = strings.Replace(scan, "@root", root, 1)

	fileList := parser.NewAst(scan)
	for _, parsers := range fileList {
		for _, fileParser := range parsers {
			if len(fileParser.Types) == 0 {
				continue
			}
			for s, goType := range fileParser.Types {
				if !goType.Doc.HasAnnotation("@Sqlite") {
					continue
				}
				orm.GenSqlite(s, goType, scan)
			}
		}
	}
}
