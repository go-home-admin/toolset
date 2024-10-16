## Swagger 文档生成

### 2024-10-16 更新

1. post的payload统一改为application/json
2. 支持多语言，新增执行参数-lang=语言标识，以`//@lang=语言 说明`声明指定语言说明
3. 优化Description显示，tag换行显示，引用对象时采用本地说明
4. 支持example定义，用`//@example=”或“//@example()`声明，前者不支持空格
5. 增加path的参数及说明，于请求声明上一行添加注释，例如：`option (http.Get) = "/user/:id";`，上一行添加：`// @query=id @lang=语言标识 @format=string @example=ABC 说明文本`，其中query是必须指定声明，format默认为int