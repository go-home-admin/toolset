package commands

import (
	"bytes"
	"fmt"
	"github.com/ctfang/command"
	"github.com/go-home-admin/toolset/console/commands/orm"
	"github.com/go-home-admin/toolset/console/commands/pgorm"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v2"
	"log"
	"os"
	"os/exec"
	"strings"
)

// OrmCommand @Bean
type OrmCommand struct{}

func (OrmCommand) Configure() command.Configure {
	return command.Configure{
		Name:        "make:orm",
		Description: "根据配置文件连接数据库, 生成orm源码",
		Input: command.Argument{
			Option: []command.ArgParam{
				{
					Name:        "config",
					Description: "配置文件",
					Default:     "@root/config/database.yaml",
				},
				{
					Name:        "out",
					Description: "输出文件",
					Default:     "@root/app/entity",
				},
			},
		},
	}
}

func (OrmCommand) Execute(input command.Input) {
	root := getRootPath()
	file := input.GetOption("config")
	file = strings.Replace(file, "@root", root, 1)
	outBase := input.GetOption("out")
	outBase = strings.Replace(outBase, "@root", root, 1)

	err := godotenv.Load(root + "/.env")
	if err != nil {
		fmt.Println(root + "/.env" + "文件不存在, 无法加载环境变量")
	}
	fileContext, _ := os.ReadFile(file)
	fileContext = SetEnv(fileContext)
	m := make(map[string]interface{})
	err = yaml.Unmarshal(fileContext, &m)
	if err != nil {
		log.Printf("配置解析错误:%v", err)
		return
	}
	connections := m["connections"].(map[interface{}]interface{})
	for s, confT := range connections {
		conf := confT.(map[interface{}]interface{})
		driver := conf["driver"]
		out := outBase + "/" + s.(string)
		switch driver {
		case "mysql":
			orm.GenMysql(s.(string), conf, out)
		case "pgsql":
			pgorm.GenSql(s.(string), conf, out)
		}

		cmd := exec.Command("go", []string{"fmt", out}...)
		var outBuffer bytes.Buffer
		cmd.Stdout = &outBuffer
		cmd.Stderr = os.Stderr
		cmd.Dir = out
		_ = cmd.Run()
	}
}

// SetEnv 对字符串内容进行替换环境变量
func SetEnv(fileContext []byte) []byte {
	str := string(fileContext)
	arr := strings.Split(str, "\n")

	for _, s := range arr {
		if strings.Index(s, " env(\"") != -1 {
			arr2 := strings.Split(s, ": ")
			if len(arr2) != 2 {
				continue
			}
			nS := arr2[1]
			st, et := GetBrackets(nS, '"', '"')
			key := nS[st+1 : et]
			nS = strings.TrimSpace(nS[et+1:])
			nS = strings.Trim(nS, ")") // 得到 ,"val" or ,val

			// 尝试获取默认值
			val := ""
			valIsStr := false
			if len(nS) > 2 && nS[0:1] == "," {
				nS = strings.TrimSpace(nS[1:])
				nS = strings.TrimSpace(nS)
				if nS[0:1] == "\"" {
					// 使用双引号括起来的就是字符串
					valIsStr = true
					st, et = GetBrackets(nS, '"', '"')
					val = nS[st+1 : et]
				} else {
					val = nS
				}
			}

			envVal, has := os.LookupEnv(key)
			if has {
				val = envVal
			}

			if !valIsStr {
				// 默认情况, 把值粘贴到yaml, 类型自动识别
				str = strings.Replace(str, s, arr2[0]+": "+val, 1)
			} else {
				// 如果有默认值, 根据默认值识别类型
				str = strings.Replace(str, s, arr2[0]+": \""+val+"\"", 1)
			}
		}
	}

	return []byte(str)
}

func GetBrackets(str string, start, end int32) (int, int) {
	var startInt, endInt int

	bCount := 0
	for i, w := range str {
		if bCount == 0 {
			if w == start {
				startInt = i
				bCount++
			}
		} else {
			switch w {
			case end:
				bCount--
				if bCount <= 0 {
					endInt = i
					return startInt, endInt
				}
			case start:
				bCount++
			}
		}
	}

	return startInt, endInt
}
