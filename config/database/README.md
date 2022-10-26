* 该目录存放表关联配置文件，
* 创建以database.yaml的connection名对应的```connection名.json```文件
* json结构：

```json
{
  "表名": [
    ...Relationship结构
  ]
}
```
* Relationship Go的结构说明：
```go
type Relationship struct {
    Type              string `json:"type"`               //关联类型：belongs_to、has_one、has_many、many2many
    Table             string `json:"table"`              //关联表名
    Alias             string `json:"alias"`              //别名（可不声明，默认用表名）
    ForeignKey        string `json:"foreign_key"`        //外键（可不声明，默认为'id'或'表名_id'）
    ReferenceKey      string `json:"reference_key"`      //引用键（可不声明，默认为'id'或'表名_id'）
    RelationshipTable string `json:"relationship_table"` //当many2many时，连接表名
    JoinForeignKey    string `json:"join_foreign_key"`   //当many2many时，本表在连接表的外键
    JoinTargetKey     string `json:"join_target_key"`    //当many2many时，关联表在连接表的外键
}
```