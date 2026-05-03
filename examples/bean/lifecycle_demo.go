package beanexample

// @Bean
type LifecycleDemo struct{}

func (LifecycleDemo) Init() {
	// 首次 NewLifecycleDemo / 等价入口时由各框架串联调用时机决定；此处仅占位生命周期方法。
}

func (LifecycleDemo) Boot() {}

func (LifecycleDemo) Exit() {}
