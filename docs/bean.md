## Bean 注解
依赖生成工具 [toolset](https://github.com/go-home-admin/toolset); 每个struct如果存在注解 @Bean 那么它就可以被依赖系统管理。使用 [`toolset make:bean`](https://github.com/go-home-admin/toolset "`toolset make:bean`")（可指定 `scan`）在目录下生成 `z_inject_gen.go`。

**可运行的标签与结构体示例**请直接看仓库源码目录 **[`examples/bean/`](../examples/bean)**（内含 `README.md` 与各 `.go` 文件），本文只保留概念说明，避免与示例代码重复维护。

如果有以下函数就是服务提供者, 即提供不是自身, 而是提供由GetBean返回的值
例如框架 config 服务提供者，在框架引导文件里定义好了，查看 [源码](https://github.com/go-home-admin/home/blob/main/bootstrap/providers/config_provider.go "源码")
```
func (*Test) GetBean(alias string) interface{} {}
```

编写好了，再使用工具生成注释对应的源码：
```shell
toolset make:bean
```

生成后 `z_inject_gen.go` 不应手改；参见 `examples/bean` 如何在业务里 `New{Type}()` 或由其它 Bean `inject` 引用。

### `inject` 写法：扁平键 vs 嵌套

`inject` 标签里逗号分左右两段（可多段默认值见工具实现）：

| 分段 | 含义 |
|------|------|
| 第一段 | 要从容器取的 **Bean 别名**，如 `config`（对应已实现 `bootstrap/providers.Bean` 的 config 提供者） |
| 第二段 | 传给该 Bean **`GetBean(...)`** 的键（点分路径由各项目 config 解析），或 **`@config(...)`** 的嵌套写法（见 `examples/bean/nested_mixed.go`） |

与「只写 `inject:""`、`GetBean("")`」的容器直引不同：**`inject:"config, key"`** 表示：先取别名为 `config` 的服务，再对其调用 **`GetBean("key")`**，返回值经类型断言后写入字段。

---

### 扁平配置注入：`inject:"config, 键"` — 值类型与指针字段

同一套 **`config` + `GetBean(键)`** 既可用于「值字段」也可用于「指针字段」，生成器对两者的**赋值方式不同**，运行时填入 `interface{}` 的具体类型也需一致，否则会断言失败。完整 struct 见 **`examples/bean/configured_flat.go`**。

生成结果在「是否对 RHS 再 unary `*`」上的差异，可对照 **`examples/bean/README.md`** 中的说明与文件表。

- **值字段（如 `int`、`string`、`pkg.Struct`）**  
  - 生成的 RHS 会先按约定断言为 **`*T`**，再在赋值处通过 unary **`*`** 解引用后赋给字段。  
  - 因此 config Bean 通常在对应键上存入 **`*T`**（例如 **`*int`**、**`*string`**），与断言一致。

- **指针字段（如 `*int`、`*pkg.Foo`）**  
  - 生成代码**不会**再给 RHS 前缀 unary `*`；整条注入表达式断言后的类型必须与字段的类型字面量一致——例如字段为 **`*int`**，则链路末端装入 `interface{}` 的动态类型也需为 **`*int`**。  
  - 常用于「可有可无」的配置：可由提供方在未配置该键时交出 **`nil`**（是否合法取决于你的 config 实现）。

与嵌套 `@config` 的差别：第二段是普通字符串键即可，**没有**以 `@` 开头；语义仍是 **`GetBean(config).(Bean).GetBean("键")`** 再断言。

**相关单测**：[`console/commands/bean_inject_assert_test.go`](../console/commands/bean_inject_assert_test.go) 校验 `iface.(…)` 断言类型字符串。

---

## 动态注入（嵌套 `@config`）

常见于「config 里有一段**字符串**（如连接名 / 子键名），再据此去另一个 Bean（如 `database`）上 `GetBean`」。示例见 **`examples/bean/nested_mixed.go`**（`DatabaseStub` 仅占位真实句柄类型）。

**生成约定**：`@config(配置路径)` **内层**在 `config` Bean 上取值时，固定断言为 **`*string`**，经 unary `*` 得到 `string`，作为**外层**第一段 Bean 的 **`GetBean(该 string)`** 的键参数；**外层**返回值的类型断言才与**字段类型**一致（如 `*DatabaseStub`）。因此内层配置路径应能在运行时提供 **`*string`**（通常指向可解析为连接名或 DSN key 的文案）。

## Bean 的生命周期与 `impl`

- `Init` / `Boot` / `Exit`：`examples/bean/lifecycle_demo.go`  
- 接口 + `impl:`：`examples/bean/impl_iface.go`（需先在容器注册实现）
