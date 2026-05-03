// Package beanexample 收录 Bean / dependency injection 的标签写法示例，供扫描生成 z_inject_gen.go。
//
// 在 toolset 仓库根目录示例：
//
//	toolset make:bean scan=@root/examples/bean
//
// 或将 @root 设为当前仓库根后仅用 scan=@root/examples/bean。
//
// 本包仅作语法参考，不依赖外部 ORM：嵌套示例使用本地的 DatabaseStub 代替 *gorm.DB。
package beanexample
