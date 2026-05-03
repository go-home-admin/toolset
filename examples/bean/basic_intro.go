package beanexample

// @Bean
type ConfigPort struct {
	// 最常见的 config + 扁平键：第二段传给 config Bean 的 GetBean("app.servers.http.port")。
	TestPort int `inject:"config, app.servers.http.port"`
}

// EchoChild 演示通过 inject:"" 从容器解析另一个 @Bean。
// @Bean
type EchoChild struct {
	Dependency *ConfigPort `inject:""`
}
