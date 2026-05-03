// gen for home toolset
package beanexample

import (
	providers "github.com/go-home-admin/home/bootstrap/providers"
)

var _ConfigPortSingle *ConfigPort
var _EchoChildSingle *EchoChild
var _ConfiguredSingle *Configured
var _ResolveFromBSingle *ResolveFromB
var _LifecycleDemoSingle *LifecycleDemo
var _MixedKindsSingle *MixedKinds
var _NestedPlaySingle *NestedPlay

func GetAllProvider() []interface{} {
	return []interface{}{
		NewConfigPort(),
		NewEchoChild(),
		NewConfigured(),
		NewResolveFromB(),
		NewLifecycleDemo(),
		NewMixedKinds(),
		NewNestedPlay(),
	}
}

func NewConfigPort() *ConfigPort {
	if _ConfigPortSingle == nil {
		_ConfigPortSingle = &ConfigPort{}
		_ConfigPortSingle.TestPort = *providers.GetBean("config").(providers.Bean).GetBean("app.servers.http.port").(*int)
		providers.AfterProvider(_ConfigPortSingle, "")
	}
	return _ConfigPortSingle
}
func NewEchoChild() *EchoChild {
	if _EchoChildSingle == nil {
		_EchoChildSingle = &EchoChild{}
		_EchoChildSingle.Dependency = NewConfigPort()
		providers.AfterProvider(_EchoChildSingle, "")
	}
	return _EchoChildSingle
}
func NewConfigured() *Configured {
	if _ConfiguredSingle == nil {
		_ConfiguredSingle = &Configured{}
		_ConfiguredSingle.MaxConn = *providers.GetBean("config").(providers.Bean).GetBean("app.db.max_conn").(*int)
		_ConfiguredSingle.OptionalPort = providers.GetBean("config").(providers.Bean).GetBean("app.optional_port").(*int)
		_ConfiguredSingle.OptionalName = providers.GetBean("config").(providers.Bean).GetBean("app.optional_label").(*string)
		providers.AfterProvider(_ConfiguredSingle, "")
	}
	return _ConfiguredSingle
}
func NewResolveFromB() *ResolveFromB {
	if _ResolveFromBSingle == nil {
		_ResolveFromBSingle = &ResolveFromB{}
		_ResolveFromBSingle.B = func() FromB {
			var temp = providers.GetBean("someRegisteredBeanAlias")
			if bean, ok := temp.(providers.Bean); ok {
				return bean.GetBean("").(FromB)
			}
			return temp.(FromB)
		}()
		providers.AfterProvider(_ResolveFromBSingle, "")
	}
	return _ResolveFromBSingle
}
func NewLifecycleDemo() *LifecycleDemo {
	if _LifecycleDemoSingle == nil {
		_LifecycleDemoSingle = &LifecycleDemo{}
		providers.AfterProvider(_LifecycleDemoSingle, "")
	}
	return _LifecycleDemoSingle
}
func NewMixedKinds() *MixedKinds {
	if _MixedKindsSingle == nil {
		_MixedKindsSingle = &MixedKinds{}
		_MixedKindsSingle.TimeoutMs = *providers.GetBean("config").(providers.Bean).GetBean("svc.timeout").(*int)
		_MixedKindsSingle.Replicas = providers.GetBean("config").(providers.Bean).GetBean("svc.replicas").(*int)
		_MixedKindsSingle.PayDSN = providers.GetBean("database").(providers.Bean).GetBean(*providers.GetBean("config").(providers.Bean).GetBean("pay.mysql_dsn").(*string)).(*DatabaseStub)
		providers.AfterProvider(_MixedKindsSingle, "")
	}
	return _MixedKindsSingle
}
func NewNestedPlay() *NestedPlay {
	if _NestedPlaySingle == nil {
		_NestedPlaySingle = &NestedPlay{}
		_NestedPlaySingle.DSNStub = providers.GetBean("database").(providers.Bean).GetBean(*providers.GetBean("config").(providers.Bean).GetBean("play.connect").(*string)).(*DatabaseStub)
		providers.AfterProvider(_NestedPlaySingle, "")
	}
	return _NestedPlaySingle
}
