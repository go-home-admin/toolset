package commands

import (
	"github.com/ctfang/command"
	"github.com/go-home-admin/toolset/parser"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
)

// BeanCommand @Bean
type BeanCommand struct{}

func (BeanCommand) Configure() command.Configure {
	return command.Configure{
		Name:        "make:bean",
		Description: "生成依赖注入的声明源代码文件, 使用@Bean注解, 和inject引入",
		Input: command.Argument{
			Has: []command.ArgParam{
				{
					Name:        "-f",
					Description: "强制更新",
				},
			},
			Option: []command.ArgParam{
				{
					Name:        "name",
					Description: "跳过目录",
					Default:     "New{name}",
				},
				{
					Name:        "scan",
					Description: "扫码目录下的源码; shell(pwd)",
					Default:     "@root",
				},
				{
					Name:        "skip",
					Description: "跳过目录",
					Default:     "@root/generate",
				},
			},
		},
	}
}

var newName = "New{name}"

func (BeanCommand) Execute(input command.Input) {
	newName = input.GetOption("name")

	root := getRootPath()
	scan := input.GetOption("scan")
	scan = strings.Replace(scan, "@root", root, 1)

	skip := make(map[string]bool)
	for _, s := range input.GetOptions("skip") {
		s = strings.Replace(s, "@root", root, 1)
		skip[s] = true
	}

	for dir, fileParsers := range parser.NewGoParserForDir(scan) {
		if _, ok := skip[dir]; ok {
			continue
		}

		bc := newBeanCache()
		for _, fileParser := range fileParsers {
			bc.name = fileParser.PackageName
			for _, goType := range fileParser.Types {
				for _, attr := range goType.Attrs {
					if attr.HasTag("inject") {
						for _, impStr := range fileParser.Imports {
							bc.imports[impStr] = impStr
						}

						break
					}
				}

				if goType.Doc.HasAnnotation("@Bean") {
					bc.structList = append(bc.structList, goType)
				}
			}
		}

		genBean(dir, bc)
	}
}

type beanCache struct {
	name       string
	imports    map[string]string
	structList []parser.GoType
}

func newBeanCache() beanCache {
	return beanCache{
		imports: map[string]string{
			"github.com/go-home-admin/home/bootstrap/services/app": "github.com/go-home-admin/home/bootstrap/services/app",
		},
		structList: make([]parser.GoType, 0),
	}
}

func genBean(dir string, bc beanCache) {
	if len(bc.structList) == 0 {
		return
	}
	context := make([]string, 0)
	context = append(context, "package "+bc.name)

	// import
	importAlias := parser.GenImportAlias(bc.imports)
	if len(importAlias) != 0 {
		context = append(context, "\nimport ("+getImportStr(bc, importAlias)+"\n)")
	}

	// Single
	context = append(context, genSingle(bc))
	// Provider
	context = append(context, genProvider(bc, importAlias))
	str := "// gen for home toolset"
	for _, s2 := range context {
		str = str + "\n" + s2
	}

	err := os.WriteFile(dir+"/z_inject_gen.go", []byte(str+"\n"), 0766)
	if err != nil {
		log.Fatal(err)
	}
}

func genSingle(bc beanCache) string {
	str := ""
	allProviderStr := "\n\treturn []interface{}{"
	for _, goType := range bc.structList {
		if goType.Doc.HasAnnotation("@Bean") {
			str = str + "\nvar " + genSingleName(goType.Name) + " *" + goType.Name
			allProviderStr += "\n\t\t" + genInitializeNewStr(goType.Name) + "(),"
		}
	}
	// 返回全部的提供商
	str += "\n\nfunc GetAllProvider() []interface{} {" + allProviderStr + "\n\t}\n}"
	return str
}

func genSingleName(s string) string {
	return "_" + s + "Single"
}

func genProvider(bc beanCache, m map[string]string) string {
	str := ""
	for _, goType := range bc.structList {
		sVar := genSingleName(goType.Name)
		if goType.Doc.HasAnnotation("@Bean") {
			str = str + "\nfunc " + genInitializeNewStr(goType.Name) + "() *" + goType.Name + " {" +
				"\n\tif " + sVar + " == nil {" + // if _provider == nil {
				"\n\t\t" + sVar + " = " + "&" + goType.Name + "{}" // provider := provider{}

			for _, attrName := range goType.AttrsSort {
				attr := goType.Attrs[attrName]
				pointer := ""
				if !attr.IsPointer() {
					pointer = "*"
				}

				for tagName := range attr.Tag {
					if tagName == "inject" {
						str = str + "\n\t\t" +
							sVar + "." + attrName + " = " + pointer + getInitializeNewFunName(attr, m)
					}
				}
			}

			constraint := m["github.com/go-home-admin/home/bootstrap/services/app"]
			str = str +
				"\n\t\t" + constraint + ".AfterProvider(" + sVar + ", \"" + goType.Doc.GetAlias() + "\")" +
				"\n\t}" +
				"\n\treturn " + sVar +
				"\n}"
		}
	}

	return str
}

func getInitializeNewFunName(k parser.GoTypeAttr, m map[string]string) string {
	alias := ""
	name := k.TypeName

	if !k.InPackage {
		a := m[k.TypeImport]
		alias = a + "."
		arr := strings.Split(k.TypeName, ".")
		name = arr[len(arr)-1]
	} else if name[0:1] == "*" {
		name = name[1:]
	}
	tag := k.Tag["inject"]
	if tag.Count() < 2 {
		return alias + genInitializeNewStr(name) + "()"
	} else {
		beanAlias := tag.Get(0)
		beanValue := tag.Get(1)

		constraint := m["github.com/go-home-admin/home/bootstrap/services/app"]

		return constraint + ".GetBean(\"" + beanAlias + "\").(" + constraint + ".Bean)" +
			".GetBean(\"" + beanValue + "\").(*" + alias + name + ")"
	}
}

// 控制对完函数名称
func genInitializeNewStr(name string) string {
	if name[0:1] == "*" {
		name = name[1:]
	}

	return strings.Replace(newName, "{name}", name, 1)
}

// 生成 import => alias
func genImportAlias(m map[string]string) map[string]string {
	aliasMapImport := make(map[string]string)
	importMapAlias := make(map[string]string)
	for _, imp := range m {
		temp := strings.Split(imp, "/")
		key := temp[len(temp)-1]

		if _, ok := aliasMapImport[key]; ok {
			for i := 1; i < 1000; i++ {
				newKey := key + strconv.Itoa(i)
				if _, ok2 := aliasMapImport[newKey]; !ok2 {
					key = newKey
					break
				}
			}
		}
		aliasMapImport[key] = imp
	}
	for s, s2 := range aliasMapImport {
		importMapAlias[s2] = s
	}

	return importMapAlias
}

// m = import => alias
func getImportStr(bc beanCache, m map[string]string) string {
	has := map[string]bool{
		"github.com/go-home-admin/home/bootstrap/services/app": true,
	}
	for _, goType := range bc.structList {
		if goType.Doc.HasAnnotation("@Bean") {
			for _, attr := range goType.Attrs {
				if !attr.InPackage {
					has[attr.TypeImport] = true
				}
			}

		}
	}
	// 删除未使用的import
	nm := make(map[string]string)
	for s, vv := range m {
		if _, ok := has[s]; ok {
			nm[s] = vv
		}
	}

	sk := sortMap(nm)
	got := ""
	for _, k := range sk {
		got += "\n\t" + nm[k] + " \"" + k + "\""
	}

	return got
}

func sortMap(m map[string]string) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
