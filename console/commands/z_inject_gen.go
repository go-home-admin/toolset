// gen for home toolset
package commands

import (
	app "github.com/go-home-admin/home/bootstrap/services/app"
)

var _BeanCommandSingle *BeanCommand
var _DevCommandSingle *DevCommand
var _OrmCommandSingle *OrmCommand
var _ProtocCommandSingle *ProtocCommand
var _RouteCommandSingle *RouteCommand
var _SwaggerCommandSingle *SwaggerCommand

func GetAllProvider() []interface{} {
	return []interface{}{
		NewBeanCommand(),
		NewDevCommand(),
		NewOrmCommand(),
		NewProtocCommand(),
		NewRouteCommand(),
		NewSwaggerCommand(),
	}
}

func NewBeanCommand() *BeanCommand {
	if _BeanCommandSingle == nil {
		_BeanCommandSingle = &BeanCommand{}
		app.AfterProvider(_BeanCommandSingle, "")
	}
	return _BeanCommandSingle
}
func NewDevCommand() *DevCommand {
	if _DevCommandSingle == nil {
		_DevCommandSingle = &DevCommand{}
		app.AfterProvider(_DevCommandSingle, "")
	}
	return _DevCommandSingle
}
func NewOrmCommand() *OrmCommand {
	if _OrmCommandSingle == nil {
		_OrmCommandSingle = &OrmCommand{}
		app.AfterProvider(_OrmCommandSingle, "")
	}
	return _OrmCommandSingle
}
func NewProtocCommand() *ProtocCommand {
	if _ProtocCommandSingle == nil {
		_ProtocCommandSingle = &ProtocCommand{}
		app.AfterProvider(_ProtocCommandSingle, "")
	}
	return _ProtocCommandSingle
}
func NewRouteCommand() *RouteCommand {
	if _RouteCommandSingle == nil {
		_RouteCommandSingle = &RouteCommand{}
		app.AfterProvider(_RouteCommandSingle, "")
	}
	return _RouteCommandSingle
}
func NewSwaggerCommand() *SwaggerCommand {
	if _SwaggerCommandSingle == nil {
		_SwaggerCommandSingle = &SwaggerCommand{}
		app.AfterProvider(_SwaggerCommandSingle, "")
	}
	return _SwaggerCommandSingle
}
