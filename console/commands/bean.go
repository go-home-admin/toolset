package commands

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/ctfang/command"
	"github.com/go-home-admin/toolset/parser"
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
					Description: "New函数别名, 如果兼容旧的项目可以设置",
					Default:     "New{name}",
				},
				{
					Name:        "scan",
					Description: "扫码目录下的源码; shell(pwd)",
					Default:     "@root",
				},
				{
					Name:        "skip",
					Description: "跳过目录；默认 @root/generate 与 @root/vendor。可 `-skip=` 重复传入，或使用逗号在一项内分隔多路径",
					Default:     "@root/generate,@root/vendor",
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
	for _, raw := range input.GetOptions("skip") {
		for _, s := range strings.Split(raw, ",") {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			s = strings.Replace(s, "@root", root, 1)
			skip[s] = true
		}
	}

	fileList := parser.NewAst(scan, skip)
	var keys []string
	for s, _ := range fileList {
		keys = append(keys, s)
	}
	sort.Strings(keys)

	for _, dir := range keys {
		fileParsers := fileList[dir]
		isSkip := false
		for s := range skip {
			if strings.Index(dir, s) != -1 {
				isSkip = true
				break
			}
		}
		if isSkip {
			continue
		}

		bc := newBeanCache()
		for _, fileParser := range fileParsers {
			bc.name = fileParser.PackageName
			keys := make([]string, 0)
			for k := range fileParser.Types {
				keys = append(keys, k)
			}
			sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
			// 排序后循环 type
			for _, k := range keys {
				goType := fileParser.Types[k]
				for _, attr := range goType.Attrs {
					if attr.HasTag("inject") || attr.HasTag("impl") {
						// 只收集使用到的 import
						bc.imports[fileParser.Imports[attr.TypeAlias]] = attr.TypeAlias
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
	name string
	// path => alias
	imports    map[string]string
	structList []parser.GoType
}

func newBeanCache() beanCache {
	return beanCache{
		imports: map[string]string{
			"github.com/go-home-admin/home/bootstrap/providers": "github.com/go-home-admin/home/bootstrap/providers",
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
	importAlias := parser.GenImportAlias(strings.ReplaceAll(dir, getRootPath(), ""), bc.name, bc.imports)
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
					} else if tagName == "impl" {
						// 使用接口 接收注入
						tmp := getInitializeNewFunName(attr, m)
						tmp = strings.ReplaceAll(tmp, "*", "")

						str = str + "\n\t\t" +
							sVar + "." + attrName + " = " + tmp
					}
				}
			}

			constraint := m["github.com/go-home-admin/home/bootstrap/providers"]
			if constraint != "" {
				constraint += "."
			}
			str = str +
				"\n\t\t" + constraint + "AfterProvider(" + sVar + ", \"" + goType.Doc.GetAlias() + "\")" +
				"\n\t}" +
				"\n\treturn " + sVar +
				"\n}"
		}
	}

	return str
}

// ifaceAssertParenBody 生成 iface.(Here) 中 Here 的类型字面量片段（不含括号）。
// 约定：运行时 GetBean / 断言得到的动态类型必须与字段赋值 RHS 的类型一致；
// - 值为 T 时 RHS 仍为 *T（外层会 *），故断言到 *T；
// - 字段类型本身已带指针（*T / **T 等）时，断言与该完整类型字面量一致并保留前缀 * 的个数，同时重写 import 别名。
func ifaceAssertParenBody(attr parser.GoTypeAttr, m map[string]string) string {
	typeRemainder := attr.TypeName
	stars := ""
	for strings.HasPrefix(typeRemainder, "*") {
		stars += "*"
		typeRemainder = typeRemainder[len("*"):]
	}

	var qualifiedRest string // 无前导星号，可能含本包多级名（尽量不破坏）
	if attr.InPackage {
		qualifiedRest = typeRemainder
	} else {
		importAlias := m[attr.TypeImport]
		if importAlias == "" {
			importAlias = attr.TypeAlias
		}
		sel := typeRemainder
		if ix := strings.LastIndexByte(sel, '.'); ix >= 0 && ix+1 < len(sel) {
			sel = sel[ix+1:]
		}
		qualifiedRest = importAlias + "." + sel
	}

	fullCanon := stars + qualifiedRest // 重写别名后的字段类型本体（无前导 unary 前缀歧义）

	if attr.IsPointer() {
		return fullCanon
	}
	return "*" + fullCanon
}

func getInitializeNewFunName(k parser.GoTypeAttr, m map[string]string) string {
	alias := ""
	name := k.TypeName

	if !k.InPackage {
		a := m[k.TypeImport]
		if a == "" {
			fmt.Println("生成注入配置错误:\n可能是识别到不明确的 import（例如别名与默认 package 推断冲突），请参考例如: redis \"github.com/go-redis/redis/v8\" 显式写清")
		} else {
			alias = a + "."
		}
		arr := strings.Split(k.TypeName, ".")
		name = arr[len(arr)-1]
	} else if name[0:1] == "*" {
		name = name[1:]
	}
	tag := k.Tag["inject"]
	if tag == "" {
		tag = k.Tag["impl"]
	}

	if tag == "" {
		return alias + genInitializeNewStr(name) + "()"
	} else {
		beanAlias := tag.Get(0)
		beanValue := tag.Get(1)

		providers := m["github.com/go-home-admin/home/bootstrap/providers"]
		if providers != "" {
			providers += "."
		}

		got := providers + "GetBean(\"" + beanAlias + "\").(" + providers + "Bean)"
		if strings.Index(beanValue, "@") != -1 {
			// 嵌套 inject: "... , @config(...)" 内层从 config Bean 取出的必须是 *string，
			// 经 unary * 得到 string，再作为外层 Bean.GetBean(键) 的第一个参数（键由配置内容动态给出）。
			// 字段类型 ifaceAssertParenBody 仅用于最外层对目标 Bean 返回值的断言。
			startTemp := strings.Index(beanValue, "(")
			beanValueNextName := beanValue[1:startTemp]
			if beanValue[len(beanValue)-1:] != ")" {
				beanValue = beanValue + ", " + tag.Get(2)
			}
			beanValueNextVal := strings.Trim(beanValue[startTemp+1:], ")")
			const nestedKeyRelayBody = "*string"
			got = got + ".GetBean(*" + providers + "GetBean(\"" + beanValueNextName + "\").(" + providers + "Bean).GetBean(\"" + beanValueNextVal + "\").(" + nestedKeyRelayBody + "))"
		} else if tag.Count() <= 2 {
			// 如果没有默认值
			if beanValue == "" {
				gotType := "*" + alias + name

				// 如果`inject:""`检查是否实现Bean接口, 没有实现就不调用
				gotFuncStr := `func() {gotType} {
			var temp = {providers}GetBean("{beanAlias}")
			if bean, ok := temp.({providers}Bean); ok {
				return bean.GetBean("").({gotType})
			}
			return temp.({gotType})
		}()`
				got = strings.ReplaceAll(gotFuncStr, "{providers}", providers)
				got = strings.Replace(got, "{beanAlias}", beanAlias, 1)
				got = strings.ReplaceAll(got, "{gotType}", gotType)
				return got
			} else {
				// 必然是实现了Bean接口
				got = got + ".GetBean(\"" + beanValue + "\")"
			}
		} else if tag.Count() == 3 {
			// 多个参数||有默认值的注入
			beanValue = beanValue + ", " + tag.Get(2)
			got = got + ".GetBean(`" + beanValue + "`)"
		}
		// 如果被注入的是接口时, 这里的*是多余的, 还不支持注入到 type interface 的变量
		// 如果有需要, 自行复制z_inject_gen.go内代码, 把开头*和类型*删除，就兼容了
		return got + ".(" + ifaceAssertParenBody(k, m) + ")"
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
	for iname, imp := range m {
		if iname != imp {
			if aliasMapImport[iname] != "" && aliasMapImport[iname] != imp {
				// 同一源码显示名映射到两条 import 路径时的后备 key（多见于 route 等生成场景）
				aliasMapImport[iname+"_2"] = imp
			} else {
				aliasMapImport[iname] = imp
			}
		} else {
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
	}
	for s, s2 := range aliasMapImport {
		importMapAlias[s2] = s
	}

	return importMapAlias
}

// cm = import => alias
func getImportStr(bc beanCache, m map[string]string) string {
	has := map[string]bool{
		"github.com/go-home-admin/home/bootstrap/providers": true,
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
		if k != "" {
			got += "\n\t" + nm[k] + " \"" + k + "\""
		}
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
