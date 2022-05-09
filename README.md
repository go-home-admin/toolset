# toolset 一个go项目工具集合
````shell
go install github.com/go-home-admin/toolset
````

![image](https://github.com/go-home-admin/toolset/blob/main/show.gif)


## 帮助命令
````shell
user@macOs path $ toolset
Usage:
  command [options] [arguments] [has]
Base Options:
  -h                     显示帮助信息
  -root                  获取项目跟路径
Available commands:
  help         帮助命令
  make:bean    生成依赖注入的声明源代码文件
  make:orm     根据配置文件连接数据库, 生成orm源码
  make:protoc  组装和执行protoc命令
  make:route   根据protoc文件定义, 生成路由信息和控制器文件
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
    -scan                = shell(pwd)
    -skip                = @root/generate
Arguments:
Option:
  -scan                  扫码目录下的源码
  -skip                  跳过目录
  -h                     显示帮助信息
  -root                  获取项目跟路径, 默认当前目录
Has:
  -f                     强制更新
Description:
   生成依赖注入的声明源代码文件, 使用@Bean注解, 和inject引入
````
