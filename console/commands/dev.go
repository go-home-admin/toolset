package commands

import (
	"fmt"
	"github.com/ctfang/command"
	"os"
	"os/exec"
)

// DevCommand @Bean
type DevCommand struct{}

func (DevCommand) Configure() command.Configure {
	return command.Configure{
		Name:        "dev",
		Description: "快速启动",
	}
}

func (DevCommand) Execute(input command.Input) {
	root := getRootPath()

	runOtherCommand("toolset", "make:protoc", "-root="+root)
	runOtherCommand("toolset", "make:route", "-root="+root)
	runOtherCommand("toolset", "make:orm", "-root="+root)
	runOtherCommand("toolset", "make:bean", "-root="+root)
	runOtherCommand("go", "run", root+"/main.go")
}

func runOtherCommand(name string, arg ...string) {
	cmd := exec.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println("cmd.Output: ", err)
		return
	}
}
