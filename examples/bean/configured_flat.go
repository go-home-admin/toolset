package beanexample

// Configured 演示同一 config Bean 下的「值类型」与「指针类型」扁平键注入。
//
// 生成器中：值字段会在 RHS 前先断言为 *T 再 unary * 赋给字段；指针字段整块表达式断言为 *T 后直接赋值。
//
// @Bean
type Configured struct {
	MaxConn      int     `inject:"config, app.db.max_conn"`
	OptionalPort *int    `inject:"config, app.optional_port"`
	OptionalName *string `inject:"config, app.optional_label"`
}
