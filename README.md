# toolset 一个go项目工具集合

1. 有 go 环境安装
````shell
go install github.com/go-home-admin/toolset@latest
````

2. 直接[下载](``https://github.com/go-home-admin/toolset/releases``)二进制文件， 放到任意 path 包含的路径下

# 展示
![image](https://github.com/go-home-admin/toolset/blob/main/show.gif)


## 帮助命令
````shell
user@macOs path $ toolset
Usage:
  command [options] [arguments] [has]
Base Options:
  -debug                 是否显示明细
  -root                  获取项目跟路径, 默认当前目录
  -h                     显示帮助信息
Available commands:
  help          帮助命令
  make          执行所有make命令, 默认参数
  make:bean     生成依赖注入的声明源代码文件, 使用@Bean注解, 和inject引入
  make:curd     生成curd基础代码, 默认使用交互输入, 便捷调用 
  make:grpc     根据protoc文件定义, 生成路grpc基础文件
  make:js       根据swagger生成js请求文件
  make:mongo    根据proto文件, 生成mongodb的orm源码
  make:orm      根据配置文件连接数据库, 生成orm源码
  make:protoc   组装和执行protoc命令
  make:route    根据protoc文件定义, 生成路由信息和控制器文件
  make:swagger  生成文档
````

## 生成ORM
````yaml
# ./config/database.yaml
connections:
  mysql:
    driver: mysql
    host: env("DB_HOST", "127.0.0.1")
    port: env("DB_PORT", "3306")
    database: env("DB_DATABASE", "home-mysql")
    username: env("DB_USERNAME", "root")
    password: env("DB_PASSWORD", "123456")
````
````shell
user@macOs path $ toolset make:orm -config=./config/database.yaml -out=your_path
````
使用, 基本上和`php` `laravel` 很类似, 如果不使用整套的`home`代码, 应该在生成目录下编写新的`NewOrmUsers`函数
````go
orm := NewOrmUsers()
user, has := orm.WhereId(1).First()
users, count := orm.WhereNickname("demo").Limit(15).Get()
fmt.Println(user, has, users, count)
````


## 生成依赖注入_
这里的原始有点像 `wire` 库, 但是不需要额外声明文件和关系, 而是使用通俗约定地生成源码，具体可以查看生成的文件`z_inject_gen.go`
````go
// Kernel @Bean
type Kernel struct {
	httpServer *services.HttpServer `inject:""`
	config     *services.Config     `inject:"config, app"`
}
````
进入目录获取传入 scan=./path; 执行命令
````shell
user@macOs path $ toolset make:bean
````

## 具体命令提示

````shell
user@macOs path $ toolset make:bean -h
Usage:
  make:bean
    -name                = New{name}
    -scan                = @root
    -skip                = @root/generate
Arguments:
Option:
  -name                  New函数别名, 如果兼容旧的项目可以设置
  -scan                  扫码目录下的源码; shell(pwd)
  -skip                  跳过目录
  -debug                 是否显示明细
  -root                  获取项目跟路径, 默认当前目录
  -h                     显示帮助信息
Has:
  -f                     强制更新
Description:
   生成依赖注入的声明源代码文件, 使用@Bean注解, 和inject引入
````


# 数据库注释使用 
1. 使用注释@type(int), 强制设置生成的go struct 属性 类型
2. 使用注释@index, 强制生成索引赋值函数


# 生成文档
定义方式
````protobuf
service Controller {
    // 发送通知文档, 不用请求
    rpc Api(ApiRequest)returns(ApiResponse){
        option (http.Post) = "/test";
        // 没有权限
        option (http.Status) = {Code: 401,Response: "TResponse"};
        // test
        option (http.Status) = {Code: 402,Response: "TTResponse"};
        // test
        option (http.Status) = {Code: 403,Response: "TTTResponse"};
    }
}
message TResponse {}
````

# 生成 js+注释 文件
````shell
user@macOs toolset % toolset make:js -h  
Usage:
  make:js
    -in                  = @root/web/swagger.json
    -out                 = @root/resources/src/api/swagger_gen.js
Arguments:
Option:
  -in                    swagger.json路径, 可本地可远程
  -out                   js文件输出路径
  -tag                   只生成指定tag的请求
  -debug                 是否显示明细
  -root                  获取项目跟路径, 默认当前目录
  -h                     显示帮助信息
Has:
Description:
   根据swagger生成js请求文件
````