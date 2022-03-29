package commands

import (
	"bytes"
	"github.com/ctfang/command"
	"github.com/go-home-admin/toolset/console/commands/orm"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v2"
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
			},
		},
	}
}

func (OrmCommand) Execute(input command.Input) {
	root := getRootPath()
	file := input.GetOption("config")
	file = strings.Replace(file, "@root", root, 1)

	err := godotenv.Load(root + "/.env")
	if err != nil {
		panic(err)
	}
	fileContext, _ := os.ReadFile(file)
	fileContext = SetEnv(fileContext)
	m := make(map[string]interface{})
	err = yaml.Unmarshal(fileContext, &m)
	if err != nil {
		panic(err)
	}

	connections := m["connections"].(map[interface{}]interface{})
	for s, confT := range connections {
		conf := confT.(map[interface{}]interface{})
		driver := conf["driver"]
		out := root + "/app/entity/" + s.(string)
		switch driver {
		case "mysql":
			orm.GenMysql(s.(string), conf, out)
		case "postgresql":

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
			key := nS[st : et+1]
			nS = nS[et+1:]
			st, et = GetBrackets(nS, '"', '"')
			val := nS[st : et+1]
			key = strings.Trim(key, "\"")
			val = strings.Trim(val, "\"")

			envVal := os.Getenv(key)
			if envVal != "" {
				val = envVal
			}

			str = strings.Replace(str, s, arr2[0]+": "+val, 1)
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
