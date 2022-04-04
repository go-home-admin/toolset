package commands

import (
	"github.com/ctfang/command"
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
	NewProtocCommand().Execute(input)
	NewRouteCommand().Execute(input)
	NewOrmCommand().Execute(input)
	NewBeanCommand().Execute(input)
}
