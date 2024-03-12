// gen for home toolset
package commands

import (
	providers "github.com/go-home-admin/home/bootstrap/providers"
)

var _BeanCommandSingle *BeanCommand
var _CurdCommandSingle *CurdCommand
var _DevCommandSingle *DevCommand
var _GrpcCommandSingle *GrpcCommand
var _JsSingle *Js
var _MongoCommandSingle *MongoCommand
var _OrmCommandSingle *OrmCommand
var _ProtocCommandSingle *ProtocCommand
var _RouteCommandSingle *RouteCommand
var _SqliteCommandSingle *SqliteCommand
var _SwaggerCommandSingle *SwaggerCommand
var _TsSingle *Ts

func GetAllProvider() []interface{} {
	return []interface{}{
		NewBeanCommand(),
		NewCurdCommand(),
		NewDevCommand(),
		NewGrpcCommand(),
		NewJs(),
		NewMongoCommand(),
		NewOrmCommand(),
		NewProtocCommand(),
		NewRouteCommand(),
		NewSqliteCommand(),
		NewSwaggerCommand(),
		NewTs(),
	}
}

func NewBeanCommand() *BeanCommand {
	if _BeanCommandSingle == nil {
		_BeanCommandSingle = &BeanCommand{}
		providers.AfterProvider(_BeanCommandSingle, "")
	}
	return _BeanCommandSingle
}
func NewCurdCommand() *CurdCommand {
	if _CurdCommandSingle == nil {
		_CurdCommandSingle = &CurdCommand{}
		providers.AfterProvider(_CurdCommandSingle, "")
	}
	return _CurdCommandSingle
}
func NewDevCommand() *DevCommand {
	if _DevCommandSingle == nil {
		_DevCommandSingle = &DevCommand{}
		providers.AfterProvider(_DevCommandSingle, "")
	}
	return _DevCommandSingle
}
func NewGrpcCommand() *GrpcCommand {
	if _GrpcCommandSingle == nil {
		_GrpcCommandSingle = &GrpcCommand{}
		providers.AfterProvider(_GrpcCommandSingle, "")
	}
	return _GrpcCommandSingle
}
func NewJs() *Js {
	if _JsSingle == nil {
		_JsSingle = &Js{}
		providers.AfterProvider(_JsSingle, "")
	}
	return _JsSingle
}
func NewMongoCommand() *MongoCommand {
	if _MongoCommandSingle == nil {
		_MongoCommandSingle = &MongoCommand{}
		providers.AfterProvider(_MongoCommandSingle, "")
	}
	return _MongoCommandSingle
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
func NewSqliteCommand() *SqliteCommand {
	if _SqliteCommandSingle == nil {
		_SqliteCommandSingle = &SqliteCommand{}
		providers.AfterProvider(_SqliteCommandSingle, "")
	}
	return _SqliteCommandSingle
}
func NewSwaggerCommand() *SwaggerCommand {
	if _SwaggerCommandSingle == nil {
		_SwaggerCommandSingle = &SwaggerCommand{}
		providers.AfterProvider(_SwaggerCommandSingle, "")
	}
	return _SwaggerCommandSingle
}
func NewTs() *Ts {
	if _TsSingle == nil {
		_TsSingle = &Ts{}
		providers.AfterProvider(_TsSingle, "")
	}
	return _TsSingle
}
