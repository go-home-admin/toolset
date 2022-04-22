package console

import (
	"github.com/ctfang/command"
	"github.com/go-home-admin/toolset/console/commands"
	"os"
	"path/filepath"
)

// Kernel @Bean
type Kernel struct{}

func (k *Kernel) Run() {
	app := command.New()
	app.AddBaseOption(command.ArgParam{
		Name:        "root",
		Description: "获取项目跟路径, 默认当前目录",
		Call: func(val string, c *command.Console) (string, bool) {
			if val == "" {
				val, _ = os.Getwd()
			}

			val, _ = filepath.Abs(val)
			commands.SetRootPath(val)
			return val, true
		},
	})
	app.AddBaseOption(command.ArgParam{
		Name:        "debug",
		Description: "是否显示明细",
		Call: func(val string, c *command.Console) (string, bool) {
			return "true", true
		},
	})

	for _, provider := range commands.GetAllProvider() {
		if v, ok := provider.(command.Command); ok {
			app.AddCommand(v)
		}
	}
	app.Run()
}

func (k *Kernel) Exit() {

}
