// gen for home toolset
package commands

import (
	providers "github.com/go-home-admin/home/bootstrap/providers"
)

var _BeanCommandSingle *BeanCommand
var _DevCommandSingle *DevCommand
var _JsSingle *Js
var _OrmCommandSingle *OrmCommand
var _ProtocCommandSingle *ProtocCommand
var _RouteCommandSingle *RouteCommand
var _SwaggerCommandSingle *SwaggerCommand

func GetAllProvider() []interface{} {
	return []interface{}{
		NewBeanCommand(),
		NewDevCommand(),
		NewJs(),
		NewOrmCommand(),
		NewProtocCommand(),
		NewRouteCommand(),
		NewSwaggerCommand(),
	}
}

func NewBeanCommand() *BeanCommand {
	if _BeanCommandSingle == nil {
		_BeanCommandSingle = &BeanCommand{}
		providers.AfterProvider(_BeanCommandSingle, "")
	}
	return _BeanCommandSingle
}
func NewDevCommand() *DevCommand {
	if _DevCommandSingle == nil {
		_DevCommandSingle = &DevCommand{}
		providers.AfterProvider(_DevCommandSingle, "")
	}
	return _DevCommandSingle
}
func NewJs() *Js {
	if _JsSingle == nil {
		_JsSingle = &Js{}
		providers.AfterProvider(_JsSingle, "")
	}
	return _JsSingle
}
func NewOrmCommand() *OrmCommand {
	if _OrmCommandSingle == nil {
		_OrmCommandSingle = &OrmCommand{}
		providers.AfterProvider(_OrmCommandSingle, "")
	}
	return _OrmCommandSingle
}
func NewProtocCommand() *ProtocCommand {
	if _ProtocCommandSingle == nil {
		_ProtocCommandSingle = &ProtocCommand{}
		providers.AfterProvider(_ProtocCommandSingle, "")
	}
	return _ProtocCommandSingle
}
func NewRouteCommand() *RouteCommand {
	if _RouteCommandSingle == nil {
		_RouteCommandSingle = &RouteCommand{}
		providers.AfterProvider(_RouteCommandSingle, "")
	}
	return _RouteCommandSingle
}
func NewSwaggerCommand() *SwaggerCommand {
	if _SwaggerCommandSingle == nil {
		_SwaggerCommandSingle = &SwaggerCommand{}
		providers.AfterProvider(_SwaggerCommandSingle, "")
	}
	return _SwaggerCommandSingle
}
