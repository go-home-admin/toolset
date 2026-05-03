package beanexample

// FromB 接口注入需在容器侧先注册具体实现 Bean，再配合 impl tag 绑定。
//
// （impl 生成的细节仍以工具与项目约定为准）
type FromB interface {
	MethodB()
}

// @Bean
type ResolveFromB struct {
	B FromB `impl:"someRegisteredBeanAlias"`
}
