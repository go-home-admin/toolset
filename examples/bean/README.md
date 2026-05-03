# Bean / inject 示例源码

路径：`examples/bean`，包名 `beanexample`。除 `z_inject_gen.go` 为 `make:bean` 生成外，其余 `.go` 为手写示例。

在项目根目录（本仓库）执行扫描，例如在 PowerShell/bash 下：

```bash
toolset make:bean scan=@root/examples/bean
```

或绝对路径等价写法。完成后各子目录将出现 `z_inject_gen.go`（若存在 `@Bean` 类型）。

## 文件对照

| 文件 | 内容 |
|------|------|
| `basic_intro.go` | config 扁平键 + `inject:""` |
| `configured_flat.go` | `int` / `*int` / `*string` 与同一条 config Bean |
| `nested_mixed.go` | `@config(...)`：内层 config 路径须为 `*string`，解引用后作外层 `GetBean` 键；`DatabaseStub` 占位句柄 |
| `lifecycle_demo.go` | `Init`/`Boot`/`Exit` |
| `impl_iface.go` | `impl` 占位说明 |
