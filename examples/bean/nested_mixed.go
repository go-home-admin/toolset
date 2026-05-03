package beanexample

// DatabaseStub 占位真实项目中的数据库句柄类型（例如 *gorm.DB）。
type DatabaseStub struct{}

// NestedPlay：inject 第一段为 database Bean；@config(play.connect) 表示在 config 上取 play.connect，
// 其值必须为 *string，解引用后的 string 再作为 database.GetBean(键) 的键；最终断言为 *DatabaseStub。
//
// @Bean
type NestedPlay struct {
	DSNStub *DatabaseStub `inject:"database, @config(play.connect)"`
}

// MixedKinds：pay.mysql_dsn 同样须由 config 解析为 *string，供 database Bean 二次查询。
//
// @Bean
type MixedKinds struct {
	TimeoutMs int           `inject:"config, svc.timeout"`
	Replicas  *int          `inject:"config, svc.replicas"`
	PayDSN    *DatabaseStub `inject:"database, @config(pay.mysql_dsn)"`
}
