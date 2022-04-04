package commands

import (
	"fmt"
	"github.com/ctfang/command"
	"log"
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
}

func runOtherCommand(name string, arg ...string) {
	cmd := exec.Command(name, arg...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("combined out:\n%s\n", string(out))
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}
	if len(out) > 0 {
		fmt.Printf("\n%s\n", string(out))
	}
}
