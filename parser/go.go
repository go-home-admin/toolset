package parser

import (
	"fmt"
	"strings"
)

/**
golang parser 非完整token实现
*/
type GoFileParser struct {
	PackageName string
	PackageDoc  string
	Imports     map[string]string
	Types       map[string]GoType
	Funds       map[string]GoFunc
}

func NewGoParserForDir(path string) []GoFileParser {
	var got []GoFileParser
	for _, file := range loadGoFiles(path) {
		gof, _ := GetFileParser(file)
		got = append(got, gof)
	}

	return got
}

func loadGoFiles(path string) []FileInfo {
	return loadFiles(path, ".go")
}

func GetFileParser(info FileInfo) (GoFileParser, error) {
	d := GoFileParser{
		PackageName: "",
		PackageDoc:  "",
		Imports:     make(map[string]string),
		Types:       make(map[string]GoType),
		Funds:       make(map[string]GoFunc),
	}

	l := getWordsWitchFile(info.path)
	lastDoc := ""
	for offset := 0; offset < len(l.list); offset++ {
		work := l.list[offset]
		// 原则上, 每个块级别的作用域必须自己处理完, 返回的偏移必须是下一个块的开始
		switch work.t {
		case wordT_line:
		case wordT_division:
		case wordT_doc:
			lastDoc = work.str
		case wordT_word:
			switch work.str {
			case "package":
				d.PackageDoc = lastDoc
				d.PackageName, offset = handlePackageName(l.list, offset)
			case "import":
				var imap map[string]string
				imap, offset = handleImports(l.list, offset)
				for k, v := range imap {
					d.Imports[k] = v
				}
			case "type":
				var imap GoType
				imap, offset = handleTypes(l.list, offset)
				imap.Doc = lastDoc
				d.Types[imap.Name] = imap
			case "func":
				var gf GoFunc
				gf, offset = handleFunds(l.list, offset)
				d.Funds[gf.Name] = gf
			case "const":
				_, offset = handleCosts(l.list, offset)
			case "var":
				_, offset = handleVars(l.list, offset)
			default:
				fmt.Println("文件块作用域似乎解析有错误", info.path, work.str, offset)
			}
		}
	}

	return d, nil
}

func handlePackageName(l []*word, offset int) (string, int) {
	newOffset := offset
	name := ""
	for i, w := range l[offset:] {
		if w.t == wordT_line {
			name = l[i-1].str
			newOffset = i
			break
		}
	}

	return name, newOffset
}

func handleImports(l []*word, offset int) (map[string]string, int) {
	newOffset := offset
	imap := make(map[string]string)
	var key, val string
	start := 0

gofer:
	for i, w := range l[offset+1:] {
		switch w.t {
		case wordT_line:
			newOffset = i + offset + 1
			switch start {
			case 0:
				break
			case 1:
				if l[newOffset+1].str == ")" {
					i = newOffset + 1
					break gofer
				}
				key, val = "", ""
			}
		case wordT_word:
			if w.str[0:1] == "\"" {
				val = w.str[1 : len(w.str)-1]

				if key == "" {
					temp := strings.Split(val, "/")
					key = temp[len(temp)-1]
				}
				imap[key] = val
			} else {
				key = w.str
			}
		case wordT_division:
			if w.str == "(" {
				start = 1
			}
		}
	}

	return imap, newOffset + 1
}

type GoType struct {
	Doc   string
	Name  string
	Attrs map[string]GoTypeAttr
}
type GoTypeAttr struct {
	Name       string
	TypeName   string
	TypeAlias  string
	TypeImport string
	Tag        map[string]string
}

// 普通指针
func (receiver GoTypeAttr) IsPointer() bool {
	return receiver.TypeName[0:1] == "*"
}

// 组装成数组, 只限name type other\n结构
func getArrGoWord(l []*word) [][]string {
	got := make([][]string, 0)
	arr := GetArrWord(l)
	for _, i := range arr {
		lis := i[len(i)-1].str
		if lis[0:1] == "`" && len(i) >= 3 {
			ty := ""
			for in := 1; in < len(i)-1; in++ {
				if i[in].t != wordT_doc {
					ty = ty + i[in].str
				}
			}
			got = append(got, []string{i[0].str, ty, lis})
		}
	}

	return got
}

// 把go结构的tag格式化成数组 source = `inject:"" json:"orm"`
func getArrGoTag(source string) [][]string {
	tagStr := source[1 : len(source)-1]
	wl := GetWords(tagStr)
	// 每三个一组
	i := 0
	got := make([][]string, 0)
	arr := make([]string, 0)
	for _, w := range wl {
		if w.t == wordT_word {
			arr = append(arr, w.str)
			i++
			if i >= 2 {
				i = 0
				got = append(got, arr)
				arr = make([]string, 0)
			}
		}
	}

	return got
}
func handleTypes(l []*word, offset int) (GoType, int) {
	newOffset := offset
	nl := l[offset:]
	got := GoType{
		Doc:   "",
		Name:  "",
		Attrs: map[string]GoTypeAttr{},
	}
	ok, off := GetLastIsIdentifier(nl, "{")
	if ok {
		// 新结构
		var i int
		got.Name, i = GetFistWordBehindStr(nl, "type")
		nl = nl[i+1:]
		st, et := GetBrackets(nl, "{", "}")
		newOffset = offset + i + et + 1
		nl := nl[st+1 : et]
		arrLn := getArrGoWord(nl)
		for _, wordAttrs := range arrLn {
			// 获取属性信息
			// TODO 当前仅支持有tag的
			if len(wordAttrs) == 3 && strings.Index(wordAttrs[2], "`") == 0 {
				attr := GoTypeAttr{
					Name:       wordAttrs[0],
					TypeName:   wordAttrs[1],
					TypeAlias:  "",
					TypeImport: "",
					Tag:        map[string]string{},
				}
				// 解析 go tag
				tagArr := getArrGoTag(wordAttrs[2])

				for _, tagStrArr := range tagArr {
					attr.Tag[tagStrArr[0]] = tagStrArr[1]
				}
				got.Attrs[attr.Name] = attr
			}
		}
	} else {
		// struct 别名
		got.Name, _ = GetFistWordBehindStr(nl, "type")
		newOffset = off + offset
	}

	return got, newOffset
}

type GoFunc struct {
	Name string
	Stu  string
}

func handleFunds(l []*word, offset int) (GoFunc, int) {
	ft := 0
	for _, w := range l[offset+1:] {
		if w.t == wordT_division && w.str == "(" {
			ft = 1
			break
		} else if w.t == wordT_word {
			break
		}
	}
	if ft == 0 {
		// 普通函数
		name, i := GetFistWordBehindStr(l[offset:], "func")

		_, et := GetBrackets(l[offset+i:], "(", ")")
		_, et = GetBrackets(l[offset+et+i:], "{", "}")
		return GoFunc{Name: name}, offset + et + i
	} else {
		// 结构函数
		_, et := GetBrackets(l[offset:], "(", ")")
		offset = offset + et
		name, _ := GetFistWord(l[offset:])
		_, et = GetBrackets(l[offset:], "(", ")")
		offset = offset + et
		_, et = GetBrackets(l[offset:], "{", "}")
		return GoFunc{Name: name}, offset + et
	}
}
func handleCosts(l []*word, offset int) (map[string]string, int) {
	ok, off := GetLastIsIdentifier(l[offset:], "(")
	if ok {
		return nil, off + offset
	}
	_, et := GetBrackets(l[offset:], "(", ")")
	return nil, offset + et
}

func handleVars(l []*word, offset int) (map[string]string, int) {
	ok, off := GetLastIsIdentifier(l[offset:], "(")
	if ok {
		return nil, off + offset
	}
	_, et := GetBrackets(l[offset:], "(", ")")
	return nil, offset + et
}
