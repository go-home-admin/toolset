// gen for home toolset
package console

import (
	providers "github.com/go-home-admin/home/bootstrap/providers"
)

var _KernelSingle *Kernel

func GetAllProvider() []interface{} {
	return []interface{}{
		NewKernel(),
	}
}

func NewKernel() *Kernel {
	if _KernelSingle == nil {
		_KernelSingle = &Kernel{}
		providers.AfterProvider(_KernelSingle, "")
	}
	return _KernelSingle
}
