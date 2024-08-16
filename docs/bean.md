## Bean 注解
依赖生成工具 [toolset](https://github.com/go-home-admin/toolset); 每个struct如果存在注解 @Bean 那么它就可以被依赖系统管理, 你可以在任意地方编写以下代码，注解依赖工具[`toolset make:bean`](https://github.com/go-home-admin/toolset "`toolset make:bean`")，这个工具可以运行任意项目下，可以不限制本框架。

```go
// @Bean
type Test struct {
	TestPort int `inject:"config, app.servers.http.port"`
}
```

如果有以下函数就是服务提供者, 即提供不是自身, 而是提供由GetBean返回的值
例如框架 config 服务提供者，在框架引导文件里定义好了，查看 [源码](https://github.com/go-home-admin/home/blob/main/bootstrap/providers/config_provider.go "源码")
```
func (*Test) GetBean(alias string) interface{} {}
```

上面代码标识，`Test ` 应该由依赖系统管理，属性TestPort 应获得一个`int`类型的配置。配置由`config`服务提供，参数是 `app.servers.http.port`，`config`服务又是由注解 @Bean("config") 管理，当然它已经在框架引导文件里定义好了，查看 [源码](https://github.com/go-home-admin/home/blob/main/bootstrap/providers/config_provider.go "源码")，你可以参考和定义更强大功能的服务提供者。编写好了，再使用工具生成注释对应的源码。

执行这个命令会扫描目录, 根据注解生成对应的源码
```shell
toolset make:bean
```
执行命令后，工具会在对应目录生成`z_inject_gen.go`，这个文件不应该手动维护；`Test `struct 就可以在其他地方使用了，只是手动`NewTest`, 也可以在别的地方使用`inject`注解注入
```go
func test() {
	NewTest()
}
// @Bean
type EchoTest struct {
	Test *Test `inject:""`
}
```
## 动态注入
为了更模块化开发，通常一个模块都有一个独立配置文件，以方便扩展功能；例如一个支付模块，需要更改自定义的数据库连接，动态注入指定配置的连接
```go
// @Bean
type Play struct {
	DB *gorm.DB `inject:"database, @config(play.connect)"`
}
```

## Bean的生命周期
```go
// @Bean
type DemoBean struct {}

func (receiver DemoBean) Init() {
	// 会在第一次NewDemoBean()时候执行
}

func (receiver DemoBean) Boot () {
	// 会在框架所有的Init执行后，再统一执行 Boot
}

func (receiver DemoBean) Exit () {
	// 应用退出时候统一执行 Exit
}
```

### 接口注入

当前为了快速解析语法树, 没有做挎包读取的功能, 无法识别接口定义, 所以需要手动编写代码, 去支持注入

~~~~go
type FromB interface {
	B()
}

// 注入 b 实现, 是不能直接支持的, 需要提前 NewB() 进行b注册到全局容器。
// 这里不要写Bean注解, 否则会报错
// 这种方式支持循环依赖
type GetB struct {
	b FromB `inject:"b"`
}

var _GetBSingle *GetB

// 手动编写代码, 去支持注入
func NewGetB() *GetB {
    if _GetBSingle == nil {
        _GetBSingle = &GetB{}
        _GetBSingle.b = func() FromB {
            var temp = providers.GetBean("b")
            if bean, ok := temp.(providers.Bean); ok {
                return bean.GetBean("").(FromB)
            }
            return temp.(FromB)
        }()
        providers.AfterProvider(_GetBSingle, "a")
    }
    return _GetBSingle
}

~~~~